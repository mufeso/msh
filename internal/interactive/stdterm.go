package interactive

import (
	"io"
	"os"

	"golang.org/x/term"
)

// StdTerminal returns the process's standard streams as a Terminal.
func StdTerminal() Terminal {
	return &stdTerminal{in: os.Stdin, out: os.Stdout, err: os.Stderr}
}

type stdTerminal struct {
	in, out, err *os.File
}

func (t *stdTerminal) Stdin() io.Reader  { return t.in }
func (t *stdTerminal) Stdout() io.Writer { return t.out }
func (t *stdTerminal) Stderr() io.Writer { return t.err }

// IsTerminal reports whether both stdin and stdout are terminals.
func (t *stdTerminal) IsTerminal() bool {
	return term.IsTerminal(int(t.in.Fd())) && term.IsTerminal(int(t.out.Fd()))
}

// MakeRaw switches stdin to raw mode: no line editing, no echo, and
// CTRL+C is delivered as a byte instead of a signal. The returned
// function restores the previous mode.
func (t *stdTerminal) MakeRaw() (restore func(), err error) {
	fd := int(t.in.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() { _ = term.Restore(fd, state) }, nil
}
