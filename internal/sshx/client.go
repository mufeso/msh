package sshx

import (
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

// Client is the part of an SSH client connection msh needs.
type Client interface {
	NewSession() (Session, error)
	Close() error
}

// Session is the part of an SSH session msh needs.
type Session interface {
	// SetStdio connects the remote process's standard streams. It must be
	// called before Start. A nil stdin means the remote side reads EOF.
	SetStdio(stdin io.Reader, stdout, stderr io.Writer)
	// StdinPipe returns a pipe connected to the remote standard input.
	// Use it instead of SetStdio's stdin when the local reader may block
	// forever (a terminal): Wait does not wait for writes to the pipe.
	StdinPipe() (io.WriteCloser, error)
	Start(cmd string) error
	Wait() error
	Signal(sig ssh.Signal) error
	Close() error

	// Interactive sessions.
	RequestPty(term string, height, width int, modes ssh.TerminalModes) error
	Shell() error
	WindowChange(height, width int) error
}

// Forwarder is what port forwarding needs from a connection. It is separate
// from Client because only real connections have it.
type Forwarder interface {
	// Dial connects from the server to addr.
	Dial(network, addr string) (net.Conn, error)
	// Listen asks the server to listen on addr.
	Listen(network, addr string) (net.Listener, error)
	// Wait blocks until the connection ends.
	Wait() error
}

// AsForwarder returns the Forwarder of c, looking through a bastion.
func AsForwarder(c Client) (Forwarder, bool) {
	if chained, ok := c.(chainedClient); ok {
		c = chained.Client
	}
	f, ok := c.(Forwarder)
	return f, ok
}

// WrapClient adapts an *ssh.Client to the Client interface.
func WrapClient(c *ssh.Client) Client {
	return clientAdapter{c}
}

type clientAdapter struct {
	c *ssh.Client
}

func (a clientAdapter) NewSession() (Session, error) {
	s, err := a.c.NewSession()
	if err != nil {
		return nil, err
	}
	return sessionAdapter{s}, nil
}

func (a clientAdapter) Close() error { return a.c.Close() }

func (a clientAdapter) Dial(network, addr string) (net.Conn, error) { return a.c.Dial(network, addr) }

func (a clientAdapter) Listen(network, addr string) (net.Listener, error) {
	return a.c.Listen(network, addr)
}

func (a clientAdapter) Wait() error { return a.c.Wait() }

// sessionAdapter embeds *ssh.Session, so Start, Wait, Signal and Close
// are promoted from it; only SetStdio needs to be written.
type sessionAdapter struct {
	*ssh.Session
}

func (a sessionAdapter) SetStdio(stdin io.Reader, stdout, stderr io.Writer) {
	a.Stdin, a.Stdout, a.Stderr = stdin, stdout, stderr
}

// Compile-time checks.
var (
	_ Client    = clientAdapter{}
	_ Forwarder = clientAdapter{}
	_ Session   = sessionAdapter{}
)
