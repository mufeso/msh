//go:build unix

package interactive

import (
	"os"
	"os/signal"
	"syscall"
)

// Resizes reports terminal size changes (SIGWINCH). stop releases the
// signal handler and closes the channel.
func (t *stdTerminal) Resizes() (<-chan struct{}, func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGWINCH)
	ch := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(ch)
		for {
			select {
			case <-sigs:
				select {
				case ch <- struct{}{}:
				default: // a resize is already pending; coalesce
				}
			case <-done:
				return
			}
		}
	}()
	return ch, func() {
		signal.Stop(sigs)
		close(done)
	}
}
