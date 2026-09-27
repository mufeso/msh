package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/sshx"
	"example.com/msh/internal/tunnel"
)

// dialTimeout limits local connections made for a forward.
const dialTimeout = 10 * time.Second

// tunnelFlags are the ssh flags for forwarding.
type tunnelFlags struct {
	local     []tunnel.Spec
	remote    []tunnel.Spec
	dynamic   []tunnel.Spec
	stdio     string // -W: host:port
	agent     bool
	noCommand bool
}

func (t *tunnelFlags) register(fs *flag.FlagSet) {
	fs.Func("L", "forward a local port: `[bind:]port:host:hostport` (like ssh -L, repeatable)", func(v string) error {
		s, err := tunnel.ParseForward(v)
		t.local = append(t.local, s)
		return err
	})
	fs.Func("R", "forward a server port back to here: `[bind:]port:host:hostport` (like ssh -R, repeatable)", func(v string) error {
		s, err := tunnel.ParseForward(v)
		t.remote = append(t.remote, s)
		return err
	})
	fs.Func("D", "run a SOCKS5 proxy here: `[bind:]port` (like ssh -D, repeatable)", func(v string) error {
		s, err := tunnel.ParseDynamic(v)
		t.dynamic = append(t.dynamic, s)
		return err
	})
	fs.Func("W", "connect stdin and stdout to `host:port` through the server (like ssh -W)", func(v string) (err error) {
		t.stdio, err = tunnel.ParseTarget(v)
		return err
	})
	fs.BoolVar(&t.agent, "A", false, "let this session use your ssh-agent; root on the server can too (like ssh -A)")
	fs.BoolVar(&t.noCommand, "N", false, "run no command, only keep the forwards open (like ssh -N)")
}

// forwarding reports whether there is anything to listen on.
func (t *tunnelFlags) forwarding() bool {
	return len(t.local)+len(t.remote)+len(t.dynamic) > 0
}

// used reports whether any of these flags was given. They all need exactly
// one connection.
func (t *tunnelFlags) used() bool {
	return t.forwarding() || t.noCommand || t.agent || t.stdio != ""
}

// tunnels runs the listeners of one connection.
type tunnels struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	cli    *cli
}

// serve runs a listener in the background.
func (r *tunnels) serve(ln net.Listener, handle tunnel.Handler) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		tunnel.Serve(r.ctx, ln, handle)
	}()
}

// report prints a failure of one forwarded connection. The tunnel stays up:
// the next connection may work.
func (r *tunnels) report(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cli.errorf("%v", err)
}

// stop closes all listeners and connections and waits for them.
func (r *tunnels) stop() {
	r.cancel()
	r.wg.Wait()
}

// warnPublic says so when a listener can be reached from other machines:
// whoever connects to it gets the access of this SSH connection.
func (c *cli) warnPublic(flagName string, s tunnel.Spec) {
	if s.Public() {
		c.errorf("warning: -%s %s: other machines can connect to this forward", flagName, s)
	}
}

// startTunnels opens every forward. If one can't be opened, none stays.
func (c *cli) startTunnels(t *tunnelFlags, opts *commonOptions, client sshx.Client) (stop func(), err error) {
	fw, ok := sshx.AsForwarder(client)
	if !ok {
		return nil, fmt.Errorf("this connection can't forward ports")
	}
	ctx, cancel := context.WithCancel(c.ctx())
	r := &tunnels{ctx: ctx, cancel: cancel, cli: c}
	defer func() {
		if err != nil {
			r.stop()
		}
	}()
	for _, s := range t.local {
		c.warnPublic("L", s)
		ln, err := net.Listen("tcp", s.BindAddr())
		if err != nil {
			return nil, fmt.Errorf("-L %s: %w", s, err)
		}
		c.verbosef(opts, "forwarding %s to %s through the server", ln.Addr(), s.Target())
		r.serve(ln, tunnel.Forward(func(context.Context) (net.Conn, error) {
			conn, err := fw.Dial("tcp", s.Target())
			if err != nil {
				return nil, fmt.Errorf("-L %s: %w", s, err)
			}
			return conn, nil
		}, r.report))
	}
	for _, s := range t.remote {
		c.warnPublic("R", s)
		ln, err := fw.Listen("tcp", s.BindAddr())
		if err != nil {
			return nil, fmt.Errorf("-R %s: %w", s, err)
		}
		c.verbosef(opts, "the server forwards %s to %s here", ln.Addr(), s.Target())
		r.serve(ln, tunnel.Forward(func(ctx context.Context) (net.Conn, error) {
			d := net.Dialer{Timeout: dialTimeout}
			conn, err := d.DialContext(ctx, "tcp", s.Target())
			if err != nil {
				return nil, fmt.Errorf("-R %s: %w", s, err)
			}
			return conn, nil
		}, r.report))
	}
	for _, s := range t.dynamic {
		c.warnPublic("D", s)
		ln, err := net.Listen("tcp", s.BindAddr())
		if err != nil {
			return nil, fmt.Errorf("-D %s: %w", s, err)
		}
		c.verbosef(opts, "SOCKS5 proxy on %s", ln.Addr())
		r.serve(ln, tunnel.Socks(func(_ context.Context, addr string) (net.Conn, error) {
			conn, err := fw.Dial("tcp", addr)
			if err != nil {
				return nil, fmt.Errorf("-D %s: %s: %w", s, addr, err)
			}
			return conn, nil
		}, r.report))
	}
	return r.stop, nil
}

// stdioForward is -W. The copy from stdin is not waited for: a terminal
// read may never return, and the end of the remote side ends the run.
func (c *cli) stdioForward(client sshx.Client, target string) exitcode.Code {
	fw, ok := sshx.AsForwarder(client)
	if !ok {
		c.errorf("this connection can't forward ports")
		return exitcode.Connection
	}
	conn, err := fw.Dial("tcp", target)
	if err != nil {
		c.errorf("-W %s: %v", target, err)
		return exitcode.Connection
	}
	defer conn.Close()
	t := c.terminal()
	go func() {
		_, _ = io.Copy(conn, t.Stdin())
		tunnel.CloseWrite(conn)
	}()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(t.Stdout(), conn)
	}()
	select {
	case <-done:
		return exitcode.OK
	case <-c.ctx().Done():
		return exitcode.Canceled
	}
}

// waitForConnection is -N: nothing runs, the forwards work until CTRL+C or
// until the connection ends.
func (c *cli) waitForConnection(client sshx.Client, server inventory.Server) exitcode.Code {
	fw, ok := sshx.AsForwarder(client)
	if !ok {
		c.errorf("this connection can't forward ports")
		return exitcode.Connection
	}
	done := make(chan error, 1)
	go func() { done <- fw.Wait() }()
	select {
	case <-c.ctx().Done():
		return exitcode.Canceled
	case err := <-done:
		c.errorf("%s: connection closed: %v", server.Name, err)
		return exitcode.Connection
	}
}
