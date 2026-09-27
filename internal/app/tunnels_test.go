package app

import (
	"context"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/sshtest"
	"golang.org/x/crypto/ssh"
)

// echoAddr starts a TCP server that returns what it receives.
func echoAddr(t *testing.T) (string, string) {
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
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return ln.Addr().String(), port
}

// freePort returns a port that was free a moment ago.
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

// background is a running msh. stop presses CTRL+C and returns the result.
type background struct {
	cancel context.CancelFunc
	done   chan struct{}
	stdout string
	stderr string
	code   exitcode.Code
}

func runBackground(t *testing.T, env map[string]string, args ...string) *background {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	b := &background{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(b.done)
		b.stdout, b.stderr, b.code = runCLICtx(t, ctx, env, args...)
	}()
	t.Cleanup(func() {
		cancel()
		<-b.done
	})
	return b
}

func (b *background) stop() {
	b.cancel()
	<-b.done
}

// dialWhenUp connects as soon as msh has opened the port.
func dialWhenUp(t *testing.T, b *background, addr string) net.Conn {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			return conn
		}
		select {
		case <-b.done:
			t.Fatalf("msh ended: code = %v, stderr = %s", b.code, b.stderr)
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatalf("nothing listens on %s", addr)
	return nil
}

// roundTrip sends text and expects it back.
func roundTrip(t *testing.T, conn net.Conn, text string) {
	t.Helper()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(text))
	if _, err := io.ReadFull(conn, got); err != nil || string(got) != text {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestLocalForward(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	echo, echoPort := echoAddr(t)
	port := freePort(t)
	args := append(fleet.credentials(), "-N", "-L", port+":127.0.0.1:"+echoPort, "tester@"+fleet.bastion.Addr)
	b := runBackground(t, nil, args...)

	conn := dialWhenUp(t, b, "127.0.0.1:"+port)
	defer conn.Close()
	roundTrip(t, conn, "through the tunnel")
	conn.Close()
	if got := fleet.bastion.Forwards(); len(got) != 1 || got[0] != echo {
		t.Errorf("forwards = %v", got)
	}
	b.stop()
	if b.code != exitcode.Canceled {
		t.Errorf("code = %v, stderr = %s", b.code, b.stderr)
	}
}

func TestLocalForwardWithCommand(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	_, echoPort := echoAddr(t)
	args := append(fleet.credentials(), "-L", freePort(t)+":127.0.0.1:"+echoPort, "tester@"+fleet.bastion.Addr, "echo", "ok")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || stdout != "ok\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestPortInUseIsAnError(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	args := append(fleet.credentials(), "-L", port+":127.0.0.1:80", "tester@"+fleet.bastion.Addr, "true")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.Connection || !contains(stderr, "-L ") || len(fleet.bastion.Commands()) != 0 {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestForwardRefusedByTheServer(t *testing.T) {
	fleet := newTestFleet(t, 1, nil) // these servers don't allow forwarding
	port := freePort(t)
	args := append(fleet.credentials(), "-N", "-L", port+":127.0.0.1:80", "tester@"+fleet.servers[0].Addr)
	b := runBackground(t, nil, args...)
	conn := dialWhenUp(t, b, "127.0.0.1:"+port)
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if n, err := conn.Read(make([]byte, 1)); n != 0 || err == nil {
		t.Errorf("read = %d, %v", n, err)
	}
	b.stop()
	if !contains(b.stderr, "port forwarding is disabled") {
		t.Errorf("stderr = %q", b.stderr)
	}
}

func TestForwardFlagErrors(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	a, c := "tester@"+fleet.servers[0].Addr, "tester@"+fleet.servers[1].Addr
	for _, args := range [][]string{
		{"-L", "nonsense", a},
		{"-N", a, "uptime"},
		{"-L", "0:127.0.0.1:80", "--host", a, c, "uptime"},
	} {
		if _, stderr, code := runCLICtx(t, context.Background(), nil, append(fleet.credentials(), args...)...); code != exitcode.Usage {
			t.Errorf("%v: code = %v, stderr = %s", args, code, stderr)
		}
	}
}

func TestPublicForwardWarns(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	args := append(fleet.credentials(), "-L", "*:0:127.0.0.1:80", "tester@"+fleet.bastion.Addr, "true")
	_, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || !contains(stderr, "other machines can connect") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func TestRemoteForward(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	_, echoPort := echoAddr(t)
	port := freePort(t)
	args := append(fleet.credentials(), "-N", "-R", port+":127.0.0.1:"+echoPort, "tester@"+fleet.bastion.Addr)
	b := runBackground(t, nil, args...)

	// The server listens here too, because it is a test server.
	conn := dialWhenUp(t, b, "127.0.0.1:"+port)
	defer conn.Close()
	roundTrip(t, conn, "back through the tunnel")
	if got := fleet.bastion.RemoteForwards(); len(got) != 1 || got[0] != "127.0.0.1:"+port {
		t.Errorf("remote forwards = %v", got)
	}
}

func TestRemoteForwardRefused(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	args := append(fleet.credentials(), "-R", "0:127.0.0.1:80", "tester@"+fleet.servers[0].Addr, "true")
	_, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.Connection || !contains(stderr, "-R ") || len(fleet.servers[0].Commands()) != 0 {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func TestDynamicForward(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	echo, echoPort := echoAddr(t)
	port := freePort(t)
	args := append(fleet.credentials(), "-N", "-D", port, "tester@"+fleet.bastion.Addr)
	b := runBackground(t, nil, args...)

	conn := dialWhenUp(t, b, "127.0.0.1:"+port)
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	p, _ := strconv.Atoi(echoPort)
	hello := []byte{5, 1, 0, 5, 1, 0, 1, 127, 0, 0, 1, byte(p >> 8), byte(p)}
	if _, err := conn.Write(hello); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 12)
	if _, err := io.ReadFull(conn, reply); err != nil || reply[3] != 0 {
		t.Fatalf("reply = %v, %v", reply, err)
	}
	roundTrip(t, conn, "by proxy")
	if got := fleet.bastion.Forwards(); len(got) != 1 || got[0] != echo {
		t.Errorf("forwards = %v", got)
	}
}

func TestStdioForward(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	_, echoPort := echoAddr(t)
	ctx := withInput(context.Background(), "ping\n")
	args := append(fleet.credentials(), "-W", "127.0.0.1:"+echoPort, "tester@"+fleet.bastion.Addr)
	stdout, stderr, code := runCLICtx(t, ctx, nil, args...)
	if code != exitcode.OK || stdout != "ping\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	args = append(fleet.credentials(), "-W", "127.0.0.1:1", "tester@"+fleet.bastion.Addr, "uptime")
	if _, _, code := runCLICtx(t, ctx, nil, args...); code != exitcode.Usage {
		t.Errorf("-W with a command: code = %v", code)
	}
}

func TestAgentForwarding(t *testing.T) {
	key := sshtest.NewKey(t)
	srv, known := passwordServer(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{key.Public()}})
	env := map[string]string{"SSH_AUTH_SOCK": sshtest.StartAgent(t, key)}
	dest := "tester@" + srv.Addr

	// Without -A the server must not see the agent.
	if stdout, stderr, code := runCLICtx(t, context.Background(), env, "--known-hosts", known, dest, "echo", "ok"); code != exitcode.OK {
		t.Fatalf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if srv.AgentKeys() != 0 {
		t.Fatal("the agent was forwarded without -A")
	}
	if stdout, stderr, code := runCLICtx(t, context.Background(), env, "-A", "--known-hosts", known, dest, "echo", "ok"); code != exitcode.OK {
		t.Fatalf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if srv.AgentKeys() != 1 {
		t.Errorf("agent keys = %d", srv.AgentKeys())
	}
}

func TestAgentForwardingNeedsAnAgent(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	args := append(fleet.credentials(), "-A", "tester@"+fleet.servers[0].Addr, "true")
	_, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.Connection || !contains(stderr, "no ssh-agent") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}
