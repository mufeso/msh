package sshx

import (
	"context"
	"time"
)

// connectClock is a timeout that can be paused. context.WithTimeout can't
// be, and the whole setup, through any number of hops, must keep one limit
// except while a person types a password.
type connectClock struct {
	timer *time.Timer
	limit time.Duration
}

type clockKey struct{}

// withConnectClock returns a context that ends with
// context.DeadlineExceeded as its cause after limit.
func withConnectClock(parent context.Context, limit time.Duration) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(parent)
	c := &connectClock{limit: limit}
	c.timer = time.AfterFunc(limit, func() { cancel(context.DeadlineExceeded) })
	return context.WithValue(ctx, clockKey{}, c), func() {
		c.timer.Stop()
		cancel(context.Canceled)
	}
}

// pauseClock stops the clock of ctx, if it has one, and returns the
// function that starts it again with the full limit.
func pauseClock(ctx context.Context) (resume func()) {
	c, ok := ctx.Value(clockKey{}).(*connectClock)
	if !ok || !c.timer.Stop() {
		return func() {}
	}
	return func() { c.timer.Reset(c.limit) }
}
