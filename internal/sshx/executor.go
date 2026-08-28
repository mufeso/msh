package sshx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/inventory"
)

// Connector opens an SSH connection to a server.
type Connector interface {
	Connect(ctx context.Context, server inventory.Server) (Client, error)
}

// Connect implements Connector. Servers with a Bastion are reached
// through that bastion; all others directly. If the bastion fails, the
// error is returned: there is no fallback to a direct connection.
func (d *Dialer) Connect(ctx context.Context, server inventory.Server) (Client, error) {
	// One timeout for the whole setup, however many hops it has.
	if d.target.cfg.ConnectTimeout > 0 {
		var stop func()
		ctx, stop = withConnectClock(ctx, d.target.cfg.ConnectTimeout)
		defer stop()
	}
	ep := EndpointFor(server)
	if server.Bastion == "" {
		c, err := d.Dial(ctx, ep)
		if err != nil {
			return nil, err
		}
		return WrapClient(c), nil
	}
	return d.connectVia(ctx, server.Bastion, ep)
}

// connectVia connects to ep through the named bastion.
func (d *Dialer) connectVia(ctx context.Context, bastionName string, ep Endpoint) (Client, error) {
	bastion, err := d.DialBastion(ctx, bastionName)
	if err != nil {
		return nil, err
	}
	conn, err := Forward(ctx, bastion, bastionName, ep)
	if err != nil {
		bastion.Close()
		return nil, err
	}
	target, err := d.Handshake(ctx, conn, ep)
	if err != nil {
		bastion.Close()
		return nil, withVia(err, bastionName)
	}
	return chainedClient{Client: WrapClient(target), bastion: bastion}, nil
}

// chainedClient is a target connection that runs through a bastion.
// Embedding Client provides NewSession; Close is overridden to close the
// target first and then the bastion connection it depends on.
type chainedClient struct {
	Client
	bastion *ssh.Client
}

func (c chainedClient) Close() error {
	err := c.Client.Close()
	if berr := c.bastion.Close(); err == nil {
		err = berr
	}
	return err
}

// Executor runs commands over SSH. It implements executor.Executor.
type Executor struct {
	Connector Connector
}

var _ executor.Executor = (*Executor)(nil)

// Run connects to server, runs req.Command and reports the outcome.
// Every connection and session it opens is closed before it returns.
func (e *Executor) Run(ctx context.Context, server inventory.Server, req executor.Request) (res executor.Result) {
	res = executor.Result{Server: server.Name, Address: server.Address, ExitCode: -1, Started: time.Now()}
	// A deferred closure can modify a named result after the return
	// statement has set it.
	defer func() { res.Duration = time.Since(res.Started) }()

	client, err := e.Connector.Connect(ctx, server)
	if err != nil {
		res.Status, res.Err = statusFor(err), err
		return res
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		res.Status, res.Err = executor.StatusError, errs.New(errs.KindRemoteCommand, server.Name, err)
		return res
	}
	defer sess.Close()

	stdout := executor.NewLimitedBuffer(req.OutputLimit)
	stderr := executor.NewLimitedBuffer(req.OutputLimit)
	var stdin io.Reader
	if req.Stdin != nil {
		stdin = bytes.NewReader(req.Stdin)
	}
	outW, errW := executor.Outputs(server, req, stdout, stderr)
	sess.SetStdio(stdin, outW, errW)

	runCtx := ctx
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	if err := sess.Start(req.Command); err != nil {
		res.Status, res.Err = executor.StatusError, errs.New(errs.KindRemoteCommand, server.Name, err)
		return res
	}

	// Session.Wait blocks and has no context parameter, so it runs in a
	// goroutine. The channel is buffered: the goroutine can always send
	// and exit, even if nobody receives.
	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()

	select {
	case err = <-done:
	case <-runCtx.Done():
		// Ask the remote process to stop (OpenSSH 7.9+ delivers signals),
		// then close the channel and the connection. Closing makes Wait
		// return, so receiving from done below cannot block forever and
		// the goroutine always finishes.
		_ = sess.Signal(ssh.SIGTERM)
		_ = sess.Close()
		_ = client.Close()
		<-done
	}

	res.Stdout, res.StdoutTruncated = stdout.String(), stdout.Truncated()
	res.Stderr, res.StderrTruncated = stderr.String(), stderr.Truncated()
	if runCtx.Err() != nil {
		interrupted(&res, ctx, runCtx, server.Name)
		return res
	}
	classifyWait(&res, server.Name, err)
	return res
}

// interrupted records why the command was stopped: the caller's context
// (user cancellation or global timeout) or the per-command timeout.
func interrupted(res *executor.Result, parent, run context.Context, host string) {
	switch {
	case errors.Is(parent.Err(), context.Canceled):
		res.Status, res.Err = executor.StatusCanceled, errs.New(errs.KindCanceled, host, parent.Err())
	case parent.Err() != nil:
		res.Status, res.Err = executor.StatusTimeout, errs.New(errs.KindGlobalTimeout, host, parent.Err())
	default:
		res.Status, res.Err = executor.StatusTimeout, errs.New(errs.KindRemoteTimeout, host, run.Err())
	}
}

// classifyWait interprets the error returned by Session.Wait.
func classifyWait(res *executor.Result, host string, err error) {
	var exitErr *ssh.ExitError
	var missing *ssh.ExitMissingError
	switch {
	case err == nil, errors.Is(err, io.EOF):
		// io.EOF: the command exited 0 without reading all of its stdin,
		// so sending the rest failed. Wait reports that error only after
		// a successful exit status, and the command did succeed.
		res.Status, res.ExitCode = executor.StatusOK, 0
	case errors.As(err, &exitErr):
		res.Status, res.ExitCode = executor.StatusFailed, exitErr.ExitStatus()
	case errors.As(err, &missing):
		res.Status = executor.StatusError
		res.Err = errs.New(errs.KindRemoteCommand, host, errors.New("the remote command ended without an exit status"))
	default:
		res.Status, res.Err = executor.StatusError, errs.New(errs.KindRemoteCommand, host, err)
	}
}

// statusFor maps a connection error to a result status.
func statusFor(err error) executor.Status {
	switch errs.KindOf(err) {
	case errs.KindCanceled:
		return executor.StatusCanceled
	case errs.KindConnectTimeout, errs.KindGlobalTimeout:
		return executor.StatusTimeout
	default:
		return executor.StatusError
	}
}

// withVia records the bastion in a target connection error.
func withVia(err error, bastion string) error {
	var e *errs.Error
	if errors.As(err, &e) && e.Role == errs.RoleTarget {
		copied := *e
		copied.Via = bastion
		return &copied
	}
	return err
}
