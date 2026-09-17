package tunnel

import (
	"context"
	"io"
	"net"
	"sync"
)

// Handler serves one accepted connection and returns when it is finished.
type Handler func(ctx context.Context, conn net.Conn)

// Dial opens the far end of a forward.
type Dial func(ctx context.Context) (net.Conn, error)

// Serve runs handle for every connection on ln until ctx ends or ln fails.
// It closes ln, and it returns only after all handlers have returned.
func Serve(ctx context.Context, ln net.Listener, handle Handler) {
	defer ln.Close()
	stop := context.AfterFunc(ctx, func() { ln.Close() })
	defer stop()

	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer conn.Close()
			// A handler may be blocked in Read: closing unblocks it.
			stop := context.AfterFunc(ctx, func() { conn.Close() })
			defer stop()
			handle(ctx, conn)
		}()
	}
}

// Forward returns a Handler that connects each connection to dial.
// report receives failures; it may be called from several goroutines.
func Forward(dial Dial, report func(error)) Handler {
	return func(ctx context.Context, conn net.Conn) {
		far, err := dial(ctx)
		if err != nil {
			if ctx.Err() == nil {
				report(err)
			}
			return
		}
		Pipe(ctx, conn, far)
	}
}

// Pipe copies between a and b until both directions end, then closes both.
func Pipe(ctx context.Context, a, b net.Conn) {
	defer a.Close()
	defer b.Close()
	stop := context.AfterFunc(ctx, func() {
		a.Close()
		b.Close()
	})
	defer stop()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		copyHalf(a, b)
	}()
	go func() {
		defer wg.Done()
		copyHalf(b, a)
	}()
	wg.Wait()
}

// copyHalf copies one direction and passes the end of data on.
func copyHalf(dst, src net.Conn) {
	_, _ = io.Copy(dst, src)
	CloseWrite(dst)
}

// CloseWrite tells the other side that no more data follows, while it can
// still send. TCP connections and SSH channels support this; for anything
// else the connection stays open until the other direction ends.
func CloseWrite(conn net.Conn) {
	if c, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = c.CloseWrite()
	}
}
