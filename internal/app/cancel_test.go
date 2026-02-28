package app

import (
	"context"
	"io"
	"os"
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
	args := append([]string{"exec", "--all", "--workers", "2", "--command-timeout", "0s", "--command", "wait"}, fleet.args()...)
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
