package interactive

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/msh/internal/inventory"
	"example.com/msh/internal/sshtest"
	"example.com/msh/internal/sshx"
	"golang.org/x/crypto/ssh"
)

// fakeTerminal records what the session did to the local terminal.
type fakeTerminal struct {
	in       io.Reader
	out, err bytes.Buffer
	tty      bool
	raw      int
	restored int
	w, h     int
	resizes  chan struct{}
}

func (f *fakeTerminal) Stdin() io.Reader        { return f.in }
func (f *fakeTerminal) Stdout() io.Writer       { return &f.out }
func (f *fakeTerminal) Stderr() io.Writer       { return &f.err }
func (f *fakeTerminal) IsTerminal() bool        { return f.tty }
func (f *fakeTerminal) Size() (int, int, error) { return f.w, f.h, nil }
func (f *fakeTerminal) Resizes() (<-chan struct{}, func()) {
	if f.resizes == nil {
		f.resizes = make(chan struct{})
	}
	return f.resizes, func() {}
}
func (f *fakeTerminal) MakeRaw() (func(), error) {
	f.raw++
	return func() { f.restored++ }, nil
}

// connect starts a test server and returns a connected client.
func connect(t *testing.T, handler sshtest.Handler) (*sshtest.Server, sshx.Client) {
	t.Helper()
	key := sshtest.NewKey(t)
	srv := sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{key.Public()}, Handler: handler})
	dir := t.TempDir()
	cfg := sshx.Config{
		IdentityFile:   sshtest.WriteKeyFile(t, dir, "id", key, ""),
		KnownHostsFile: sshtest.WriteKnownHosts(t, dir, srv.KnownHostsLine()),
		ConnectTimeout: 5 * time.Second,
	}
	auth, err := sshx.NewAuth(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { auth.Close() })
	d, err := sshx.NewDialer(cfg, auth)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, _ := net.SplitHostPort(srv.Addr)
	port, _ := strconv.Atoi(portStr)
	client, err := d.Connect(context.Background(), inventory.Server{Name: "s", Address: host, Port: port, User: "u"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return srv, client
}

func TestRawModeOnlyOnTerminals(t *testing.T) {
	for _, tty := range []bool{true, false} {
		srv, client := connect(t, nil)
		term := &fakeTerminal{in: strings.NewReader("exit 0\n"), tty: tty}
		if _, err := Run(context.Background(), client, term, Options{Term: "xterm"}); err != nil {
			t.Fatal(err)
		}
		wantPTY := 0
		if tty {
			wantPTY = 1
		}
		if len(srv.PTYs()) != wantPTY || term.raw != wantPTY || term.restored != term.raw {
			t.Errorf("tty=%v: ptys=%d raw=%d restored=%d", tty, len(srv.PTYs()), term.raw, term.restored)
		}
	}
}

func TestRemoteExitDoesNotWaitForLocalInput(t *testing.T) {
	_, client := connect(t, nil)
	pr, pw := io.Pipe() // never written: like a terminal nobody types into
	defer pw.Close()
	term := &fakeTerminal{in: pr, tty: false}
	done := make(chan int, 1)
	go func() {
		status, _ := Run(context.Background(), client, term, Options{Command: "echo bye; exit 3"})
		done <- status
	}()
	select {
	case status := <-done:
		if status != 3 {
			t.Errorf("status = %d, want 3", status)
		}
	case <-time.After(3 * time.Second):
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		t.Fatalf("Run waited for local stdin after the remote command exited\n%s", buf[:n])
	}
	if !strings.Contains(term.out.String(), "bye") {
		t.Errorf("stdout = %q", term.out.String())
	}
}

func TestLocalEOFEndsRemoteShell(t *testing.T) {
	_, client := connect(t, nil)
	term := &fakeTerminal{in: strings.NewReader("echo first\necho second\n"), tty: false}
	status, err := Run(context.Background(), client, term, Options{})
	if err != nil || status != 0 {
		t.Fatalf("status = %d, err = %v", status, err)
	}
	if term.out.String() != "first\nsecond\n" {
		t.Errorf("stdout = %q", term.out.String())
	}
}

func TestNoTTYKeepsStderrSeparate(t *testing.T) {
	srv, client := connect(t, nil)
	term := &fakeTerminal{in: strings.NewReader(""), tty: true}
	status, err := Run(context.Background(), client, term, Options{Command: "echo out; echo err >&2", NoTTY: true})
	if err != nil || status != 0 {
		t.Fatalf("status = %d, err = %v", status, err)
	}
	if len(srv.PTYs()) != 0 || term.raw != 0 {
		t.Error("a PTY or raw mode was used despite NoTTY")
	}
	if term.out.String() != "out\n" || term.err.String() != "err\n" {
		t.Errorf("stdout = %q, stderr = %q", term.out.String(), term.err.String())
	}
}

func TestPTYUsesTerminalSize(t *testing.T) {
	srv, client := connect(t, nil)
	term := &fakeTerminal{in: strings.NewReader("exit\n"), tty: true, w: 132, h: 43}
	if _, err := Run(context.Background(), client, term, Options{Term: "screen"}); err != nil {
		t.Fatal(err)
	}
	ptys := srv.PTYs()
	if len(ptys) != 1 || ptys[0].Width != 132 || ptys[0].Height != 43 || ptys[0].Term != "screen" {
		t.Errorf("pty = %+v", ptys)
	}
}

func TestPTYDefaultSize(t *testing.T) {
	srv, client := connect(t, nil)
	term := &fakeTerminal{in: strings.NewReader("exit\n"), tty: true}
	if _, err := Run(context.Background(), client, term, Options{Term: "xterm"}); err != nil {
		t.Fatal(err)
	}
	if p := srv.PTYs(); len(p) != 1 || p[0].Width != 80 || p[0].Height != 24 {
		t.Errorf("pty = %+v", p)
	}
}

func TestResizeSendsWindowChange(t *testing.T) {
	srv, client := connect(t, sshtest.Blocking(""))
	term := &fakeTerminal{in: strings.NewReader(""), tty: true, w: 80, h: 24, resizes: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		Run(ctx, client, term, Options{Term: "xterm", Command: "wait"})
	}()
	// Wait for the session, then "resize the window".
	deadline := time.Now().Add(3 * time.Second)
	for len(srv.PTYs()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	term.w, term.h = 200, 50
	term.resizes <- struct{}{}
	for len(srv.Resizes()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if r := srv.Resizes(); len(r) != 1 || r[0] != [2]int{200, 50} {
		t.Errorf("window changes = %v", r)
	}
}

func TestCancellationHangsUpAndRestores(t *testing.T) {
	srv, client := connect(t, sshtest.Blocking("ready\n"))
	term := &fakeTerminal{in: strings.NewReader(""), tty: true}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, err := Run(ctx, client, term, Options{Term: "xterm", Command: "wait"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Error("cancellation was not prompt")
	}
	if term.raw != 1 || term.restored != 1 {
		t.Errorf("raw = %d, restored = %d: the terminal must be restored", term.raw, term.restored)
	}
	if sig := srv.Signals(); len(sig) == 0 || sig[0] != "HUP" {
		t.Errorf("server signals = %v, want HUP", sig)
	}
	if !srv.WaitIdle(2 * time.Second) {
		t.Error("session still open")
	}
}
