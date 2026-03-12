// Package signals turns SIGINT (CTRL+C), SIGTERM and SIGHUP into context
// cancellation.
//
// The first signal cancels the root context: every operation derived
// from it (connections, sessions, workers, local processes) stops. A
// second signal exits immediately, for the case where something does not
// stop.
package signals

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// ErrInterrupted is the cancellation cause after a signal.
var ErrInterrupted = errors.New("interrupted")

// ForceExitCode is used when a second signal arrives (128 + SIGINT).
const ForceExitCode = 130

// Context returns a context canceled by SIGINT or SIGTERM. stop releases
// the signal handler and must be called when the program ends.
func Context(parent context.Context, stderr io.Writer) (ctx context.Context, stop func()) {
	// Buffered: signal.Notify does not block, so an unbuffered channel
	// could miss a signal.
	sigs := make(chan os.Signal, 2)
	// SIGHUP: the terminal was closed. In raw mode (interactive sessions)
	// CTRL+C does not produce SIGINT; the keystroke goes to the server.
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	ctx, stopWatch := Watch(parent, sigs, stderr, os.Exit)
	return ctx, func() {
		signal.Stop(sigs)
		stopWatch()
	}
}

// Watch implements Context with an injectable signal channel and exit
// function, so it can be tested without real signals.
func Watch(parent context.Context, sigs <-chan os.Signal, stderr io.Writer, exit func(int)) (ctx context.Context, stop func()) {
	ctx, cancel := context.WithCancelCause(parent)
	done := make(chan struct{})
	go func() {
		select {
		case sig := <-sigs:
			fmt.Fprintf(stderr, "\nmsh: received %v, stopping all work (press CTRL+C again to force exit)\n", sig)
			cancel(ErrInterrupted)
		case <-done:
			return
		}
		select {
		case sig := <-sigs:
			fmt.Fprintf(stderr, "msh: received %v again, exiting immediately\n", sig)
			exit(ForceExitCode)
		case <-done:
		}
	}()
	return ctx, func() {
		close(done)
		cancel(nil)
	}
}

// Interrupted reports whether ctx was canceled by a signal.
func Interrupted(ctx context.Context) bool {
	return errors.Is(context.Cause(ctx), ErrInterrupted)
}
