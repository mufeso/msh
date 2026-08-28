package sshx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConnectClockExpires(t *testing.T) {
	ctx, stop := withConnectClock(context.Background(), 10*time.Millisecond)
	defer stop()
	<-ctx.Done()
	if !errors.Is(context.Cause(ctx), context.DeadlineExceeded) {
		t.Errorf("cause = %v", context.Cause(ctx))
	}
}

func TestConnectClockPauses(t *testing.T) {
	ctx, stop := withConnectClock(context.Background(), 30*time.Millisecond)
	defer stop()
	resume := pauseClock(ctx)
	time.Sleep(80 * time.Millisecond)
	if ctx.Err() != nil {
		t.Fatal("the clock ran while paused")
	}
	resume()
	<-ctx.Done()
	if !errors.Is(context.Cause(ctx), context.DeadlineExceeded) {
		t.Errorf("cause = %v", context.Cause(ctx))
	}
}
