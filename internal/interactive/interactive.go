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
	// Size returns the terminal's width and height in characters.
	Size() (width, height int, err error)
	// Resizes delivers a value whenever the size changes. stop ends the
	// watching and closes the channel.
	Resizes() (changes <-chan struct{}, stop func())
}

// size returns the terminal size or 80x24 if it cannot be determined.
func size(t Terminal) (width, height int) {
	w, h, err := t.Size()
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

// Options describe the session.
type Options struct {
	Term    string // value for $TERM on the server
	Command string // remote command; "" starts the login shell
	NoTTY   bool   // never request a pseudo-terminal (keeps stderr separate)
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

	// With a PTY, the remote side writes stdout and stderr to the same
	// terminal. Without one, stderr arrives on its own stream.
	tty := t.IsTerminal() && !opts.NoTTY
	if tty {
		modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 38400, ssh.TTY_OP_OSPEED: 38400}
		w, h := size(t)
		if err := sess.RequestPty(opts.Term, h, w, modes); err != nil {
			return -1, err
		}
	}
	// Standard input is copied by our own goroutine through a pipe. If it
	// were passed to SetStdio, Session.Wait would wait for that copy, and
	// a terminal Read only returns on the next keypress. This goroutine is
	// not waited for: it ends on EOF, on a write error after the session
	// closed, or when msh exits.
	sess.SetStdio(nil, t.Stdout(), t.Stderr())
	stdin, err := sess.StdinPipe()
	if err != nil {
		return -1, err
	}
	go func() {
		_, _ = io.Copy(stdin, t.Stdin())
		_ = stdin.Close() // local EOF: the remote side reads EOF too
	}()
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

	if tty {
		changes, stop := t.Resizes()
		defer stop()
		go forwardResizes(t, sess, changes)
	}

	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()
	select {
	case err := <-done:
		return exitStatus(err)
	case <-ctx.Done():
		// Like a closed terminal: hang up the remote side, close the
		// session (which makes Wait return), and wait for the goroutine.
		_ = sess.Signal(ssh.SIGHUP)
		_ = sess.Close()
		<-done
		return -1, ctx.Err()
	}
}

// forwardResizes sends the new size to the server after every change.
// It returns when the changes channel is closed (stop was called).
func forwardResizes(t Terminal, sess sshx.Session, changes <-chan struct{}) {
	for range changes {
		w, h := size(t)
		// Errors are ignored: the session may be ending.
		_ = sess.WindowChange(h, w)
	}
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
