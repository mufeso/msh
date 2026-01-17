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

// Connect implements Connector with a direct connection.
func (d *Dialer) Connect(ctx context.Context, server inventory.Server) (Client, error) {
	c, err := d.Dial(ctx, EndpointFor(server))
	if err != nil {
		return nil, err
	}
	return WrapClient(c), nil
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
	sess.SetStdio(stdin, stdout, stderr)

	if err := sess.Start(req.Command); err != nil {
		res.Status, res.Err = executor.StatusError, errs.New(errs.KindRemoteCommand, server.Name, err)
		return res
	}
	err = sess.Wait()
	res.Stdout, res.StdoutTruncated = stdout.String(), stdout.Truncated()
	res.Stderr, res.StderrTruncated = stderr.String(), stderr.Truncated()
	classifyWait(&res, server.Name, err)
	return res
}

// classifyWait interprets the error returned by Session.Wait.
func classifyWait(res *executor.Result, host string, err error) {
	var exitErr *ssh.ExitError
	var missing *ssh.ExitMissingError
	switch {
	case err == nil:
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
