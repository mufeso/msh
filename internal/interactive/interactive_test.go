package interactive

import (
	"bytes"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/inventory"
	"example.com/msh/internal/sshtest"
	"example.com/msh/internal/sshx"
)

// fakeTerminal records what the session did to the local terminal.
type fakeTerminal struct {
	in       io.Reader
	out, err bytes.Buffer
	tty      bool
	raw      int
	restored int
}

func (f *fakeTerminal) Stdin() io.Reader  { return f.in }
func (f *fakeTerminal) Stdout() io.Writer { return &f.out }
func (f *fakeTerminal) Stderr() io.Writer { return &f.err }
func (f *fakeTerminal) IsTerminal() bool  { return f.tty }
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
