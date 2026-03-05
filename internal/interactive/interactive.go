// Package interactive runs interactive SSH sessions: a remote login shell
// or a remote command that needs a terminal (top, vi, "podman exec -it").
package interactive

import (
	"context"
	"errors"
	"io"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/sshx"
)

// Terminal is the local side of an interactive session.
type Terminal interface {
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer

	// IsTerminal reports whether the streams are an interactive terminal.
	IsTerminal() bool
	// MakeRaw puts the terminal into raw mode and returns a function
	// that restores the previous mode.
	MakeRaw() (restore func(), err error)
}

// Options describe the session.
type Options struct {
	Term    string // value for $TERM on the server
	Command string // remote command; "" starts the login shell
}

// Run opens a session on client, requests a pseudo-terminal, connects the
// terminal's streams and starts the remote login shell or command. It
// blocks until the session ends and returns the remote exit status.
func Run(ctx context.Context, client sshx.Client, t Terminal, opts Options) (int, error) {
	sess, err := client.NewSession()
	if err != nil {
		return -1, err
	}
	defer sess.Close()

	tty := t.IsTerminal()
	if tty {
		modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 38400, ssh.TTY_OP_OSPEED: 38400}
		if err := sess.RequestPty(opts.Term, 24, 80, modes); err != nil {
			return -1, err
		}
	}
	sess.SetStdio(t.Stdin(), t.Stdout(), t.Stderr())
	if tty {
		restore, err := t.MakeRaw()
		if err != nil {
			return -1, err
		}
		// Whatever happens next, the user gets their terminal back.
		defer restore()
	}
	if opts.Command == "" {
		err = sess.Shell()
	} else {
		err = sess.Start(opts.Command)
	}
	if err != nil {
		return -1, err
	}
	return exitStatus(sess.Wait())
}

// exitStatus converts the result of Session.Wait into an exit status.
func exitStatus(err error) (int, error) {
	var exitErr *ssh.ExitError
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &exitErr):
		return exitErr.ExitStatus(), nil
	default:
		return -1, err
	}
}
