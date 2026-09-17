package tunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// echoServer answers with what it receives, until the client half-closes.
func echoServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				io.Copy(conn, conn)
			}()
		}
	}()
	return ln.Addr().String()
}

// listen returns a listener and its address.
func listen(t *testing.T) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln, ln.Addr().String()
}

func dialTo(addr string) Dial {
	return func(context.Context) (net.Conn, error) { return net.Dial("tcp", addr) }
}

func TestForwardCarriesDataAndHalfClose(t *testing.T) {
	echo := echoServer(t)
	ln, addr := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		Serve(ctx, ln, Forward(dialTo(echo), func(err error) { t.Error(err) }))
	}()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write([]byte("hello"))
	conn.(*net.TCPConn).CloseWrite() // done sending, still reading
	got, err := io.ReadAll(conn)
	if err != nil || string(got) != "hello" {
		t.Errorf("got %q, %v", got, err)
	}
	cancel()
	<-done
}

func TestCancelStopsActiveConnections(t *testing.T) {
	ln, addr := listen(t)
	silent, silentAddr := listen(t) // accepts and never answers
	defer silent.Close()
	go func() {
		for {
			conn, err := silent.Accept()
			if err != nil {
				return
			}
			defer conn.Close()
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		Serve(ctx, ln, Forward(dialTo(silentAddr), func(error) {}))
	}()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write([]byte("x"))
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not stop")
	}
}

func TestDialFailureIsReported(t *testing.T) {
	ln, addr := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var got error
	reported := make(chan struct{})
	dial := func(context.Context) (net.Conn, error) { return nil, errors.New("refused") }
	go Serve(ctx, ln, Forward(dial, func(err error) {
		mu.Lock()
		defer mu.Unlock()
		got = err
		close(reported)
	}))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	<-reported
	if n, err := conn.Read(make([]byte, 1)); n != 0 || err == nil {
		t.Errorf("the local side must see the connection close: %d, %v", n, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got == nil || got.Error() != "refused" {
		t.Errorf("got %v", got)
	}
}
