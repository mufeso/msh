package app

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/signals"
	"example.com/msh/internal/sshtest"
)

func TestCtrlCStopsEverything(t *testing.T) {
	fleet := newTestFleet(t, 4, sshtest.Blocking(""))
	sigs := make(chan os.Signal, 1)
	ctx, stop := signals.Watch(context.Background(), sigs, io.Discard, func(int) {})
	defer stop()
	go func() {
		time.Sleep(300 * time.Millisecond)
		sigs <- os.Interrupt // the user presses CTRL+C
	}()

	start := time.Now()
	args := append([]string{"exec", "--all", "--blocks", "--workers", "2", "--command-timeout", "0s", "--command", "wait"}, fleet.args()...)
	stdout, _, code := runCLICtx(t, ctx, nil, args...)
	if code != exitcode.Canceled {
		t.Errorf("code = %v, want canceled (130)", code)
	}
	if time.Since(start) > 5*time.Second {
		t.Error("cancellation was not prompt")
	}
	if n := strings.Count(stdout, ": canceled ("); n != 4 {
		t.Errorf("%d canceled results, want 4:\n%s", n, stdout)
	}
	for i, s := range fleet.servers {
		if !s.WaitIdle(2 * time.Second) {
			t.Errorf("s%d: session still open after CTRL+C", i+1)
		}
	}
}

func TestNoGoroutineLeakAfterCancel(t *testing.T) {
	fleet := newTestFleet(t, 3, sshtest.Blocking(""))
	before := runtime.NumGoroutine()
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		args := append([]string{"exec", "--all", "--command-timeout", "0s", "--command", "wait"}, fleet.args()...)
		runCLICtx(t, ctx, nil, args...)
		cancel()
	}
	for _, s := range fleet.servers {
		s.WaitIdle(2 * time.Second)
	}
	deadline := time.Now().Add(3 * time.Second)
	for runtime.NumGoroutine() > before+2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Errorf("goroutines before %d, after %d", before, after)
	}
}
