package interactive

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// ReadPassword shows prompt and reads a line without echo.
type ReadPassword func(ctx context.Context, prompt string) (string, error)

// PasswordReader returns a reader for the controlling terminal, or false
// without one (cron, CI). It uses /dev/tty because stdin may be a pipe that
// belongs to the remote command.
func PasswordReader() (ReadPassword, bool) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, false
	}
	tty.Close()
	return readFromTTY, true
}

func readFromTTY(ctx context.Context, prompt string) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", err
	}
	defer tty.Close()
	fd := int(tty.Fd())
	state, err := term.GetState(fd)
	if err != nil {
		return "", err
	}
	fmt.Fprint(tty, prompt)

	type result struct {
		line []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		line, err := term.ReadPassword(fd)
		done <- result{line, err}
	}()
	select {
	case r := <-done:
		fmt.Fprintln(tty)
		if r.err == nil && len(r.line) == 0 {
			return "", errors.New("no password given")
		}
		return string(r.line), r.err
	case <-ctx.Done():
		// The reading goroutine is stuck in read(2) and can't restore
		// the terminal; without this the shell would stay without echo.
		_ = term.Restore(fd, state)
		fmt.Fprintln(tty)
		return "", ctx.Err()
	}
}
