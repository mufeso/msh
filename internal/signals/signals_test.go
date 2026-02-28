package signals

import (
	"bytes"
	"context"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer safe for concurrent use.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestFirstSignalCancelsSecondExits(t *testing.T) {
	sigs := make(chan os.Signal, 2)
	var stderr syncBuffer
	exited := make(chan int, 1)
	ctx, stop := Watch(context.Background(), sigs, &stderr, func(code int) { exited <- code })
	defer stop()

	sigs <- os.Interrupt
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context not canceled by the first signal")
	}
	if !Interrupted(ctx) {
		t.Errorf("cause = %v", context.Cause(ctx))
	}
	sigs <- os.Interrupt
	select {
	case code := <-exited:
		if code != 130 {
			t.Errorf("exit code = %d", code)
		}
	case <-time.After(time.Second):
		t.Fatal("second signal did not force an exit")
	}
	if !bytes.Contains([]byte(stderr.String()), []byte("stopping all work")) {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestStopWithoutSignal(t *testing.T) {
	ctx, stop := Watch(context.Background(), make(chan os.Signal), &syncBuffer{}, func(int) { t.Error("exit called") })
	stop()
	if Interrupted(ctx) {
		t.Error("stop must not look like an interruption")
	}
}

func TestRealSignal(t *testing.T) {
	var stderr syncBuffer
	ctx, stop := Context(context.Background(), &stderr)
	defer stop()
	// Deliver a real SIGINT to this test process. Because Context
	// registered a handler, the process is not killed.
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("real SIGINT did not cancel the context")
	}
}
