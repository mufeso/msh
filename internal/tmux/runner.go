package tmux

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Runner runs tmux commands.
type Runner interface {
	// Output runs "tmux ARGS..." and returns its trimmed standard output.
	Output(ctx context.Context, args ...string) (string, error)
	// Attach runs "tmux ARGS..." connected to the user's terminal, for
	// attach-session, and returns when the user detaches or the session
	// ends.
	Attach(ctx context.Context, args ...string) error
}

// ExecRunner runs the real tmux binary. Arguments are passed as a slice
// to exec.CommandContext: no shell parses them.
type ExecRunner struct {
	Path   string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Output implements Runner.
func (r ExecRunner) Output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, r.Path, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("tmux %s: %s", args[0], msg)
	}
	return strings.TrimSpace(string(out)), nil
}

// Attach implements Runner.
func (r ExecRunner) Attach(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, r.Path, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = r.Stdin, r.Stdout, r.Stderr
	return cmd.Run()
}

// Version returns the output of "tmux -V", e.g. "tmux 3.4".
func Version(ctx context.Context, r Runner) (string, error) {
	return r.Output(ctx, "-V")
}
