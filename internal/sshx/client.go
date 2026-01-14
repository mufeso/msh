package sshx

import (
	"io"

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
	Start(cmd string) error
	Wait() error
	Signal(sig ssh.Signal) error
	Close() error
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
	_ Client  = clientAdapter{}
	_ Session = sessionAdapter{}
)
