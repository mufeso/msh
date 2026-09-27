package tunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// startSocks runs a proxy whose dial records the address.
func startSocks(t *testing.T, dial DialAddr) string {
	t.Helper()
	ln, addr := listen(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		Serve(ctx, ln, Socks(dial, func(error) {}))
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return addr
}

// exchange sends bytes and reads n bytes back.
func exchange(t *testing.T, conn net.Conn, send []byte, n int) []byte {
	t.Helper()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(send); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, n)
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSocksConnect(t *testing.T) {
	echo := echoServer(t)
	seen := make(chan string, 3)
	proxy := startSocks(t, func(_ context.Context, addr string) (net.Conn, error) {
		seen <- addr
		return net.Dial("tcp", echo)
	})
	tests := []struct {
		name    string
		address []byte
		want    string
	}{
		{"IPv4", []byte{addrIPv4, 192, 0, 2, 1, 0, 80}, "192.0.2.1:80"},
		{"name", append(append([]byte{addrName, 11}, "example.com"...), 1, 187), "example.com:443"},
		{"IPv6", append(append([]byte{addrIPv6}, net.ParseIP("2001:db8::1")...), 0, 22), "[2001:db8::1]:22"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := net.Dial("tcp", proxy)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if got := exchange(t, conn, []byte{5, 1, 0}, 2); got[1] != noAuth {
				t.Fatalf("greeting = %v", got)
			}
			request := append([]byte{5, cmdConnect, 0}, tt.address...)
			if got := exchange(t, conn, request, 10); got[1] != replyOK {
				t.Fatalf("reply = %v", got)
			}
			if got := <-seen; got != tt.want {
				t.Errorf("address = %q", got)
			}
			if got := exchange(t, conn, []byte("hi"), 2); string(got) != "hi" {
				t.Errorf("data = %q", got)
			}
		})
	}
}

func TestSocksRefusals(t *testing.T) {
	proxy := startSocks(t, func(context.Context, string) (net.Conn, error) { return nil, errors.New("no") })
	dial := func() net.Conn {
		conn, err := net.Dial("tcp", proxy)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })
		return conn
	}
	if got := exchange(t, dial(), []byte{5, 1, 2}, 2); got[1] != noMethod {
		t.Errorf("password only: %v", got)
	}
	conn := dial()
	exchange(t, conn, []byte{5, 1, 0}, 2)
	if got := exchange(t, conn, []byte{5, 2, 0, addrIPv4}, 10); got[1] != replyBadCommand {
		t.Errorf("BIND: %v", got)
	}
	conn = dial()
	exchange(t, conn, []byte{5, 1, 0}, 2)
	if got := exchange(t, conn, []byte{5, 1, 0, 9}, 10); got[1] != replyBadAddress {
		t.Errorf("bad address type: %v", got)
	}
	conn = dial()
	exchange(t, conn, []byte{5, 1, 0}, 2)
	if got := exchange(t, conn, []byte{5, 1, 0, addrIPv4, 192, 0, 2, 1, 0, 80}, 10); got[1] != replyUnreachable {
		t.Errorf("dial failure: %v", got)
	}
	conn = dial() // not SOCKS at all: closed without an answer
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
	if n, err := conn.Read(make([]byte, 1)); n != 0 || err == nil {
		t.Errorf("read = %d, %v", n, err)
	}
}
