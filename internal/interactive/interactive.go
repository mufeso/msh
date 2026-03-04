// Package interactive runs interactive SSH sessions: a remote login shell
// or a remote command that needs a terminal (top, vi, "podman exec -it").
package interactive

import (
	"context"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/sshx"
)

// Terminal is the local side of an interactive session.
type Terminal interface {
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
}

// StdTerminal returns the process's standard streams.
func StdTerminal() Terminal { return stdTerminal{} }

type stdTerminal struct{}

func (stdTerminal) Stdin() io.Reader  { return os.Stdin }
func (stdTerminal) Stdout() io.Writer { return os.Stdout }
func (stdTerminal) Stderr() io.Writer { return os.Stderr }

// Options describe the session.
type Options struct {
	Term string // value for $TERM on the server
}

// Run opens a session on client, requests a pseudo-terminal, connects the
// terminal's streams and starts the remote login shell. It blocks until
// the session ends and returns the remote exit status.
func Run(ctx context.Context, client sshx.Client, t Terminal, opts Options) (int, error) {
	sess, err := client.NewSession()
	if err != nil {
		return -1, err
	}
	defer sess.Close()

	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 38400, ssh.TTY_OP_OSPEED: 38400}
	if err := sess.RequestPty(opts.Term, 24, 80, modes); err != nil {
		return -1, err
	}
	sess.SetStdio(t.Stdin(), t.Stdout(), t.Stderr())
	if err := sess.Shell(); err != nil {
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
