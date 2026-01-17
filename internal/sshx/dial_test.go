package sshx

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/sshtest"
)

// testEnv is a server plus everything a client needs to reach it.
type testEnv struct {
	server *sshtest.Server
	key    sshtest.Key
	cfg    Config
	dir    string
}

func newTestEnv(t *testing.T, opts sshtest.Options) *testEnv {
	t.Helper()
	key := sshtest.NewKey(t)
	if opts.AuthorizedKeys == nil {
		opts.AuthorizedKeys = []ssh.PublicKey{key.Public()}
	}
	srv := sshtest.NewServer(t, opts)
	dir := t.TempDir()
	return &testEnv{
		server: srv,
		key:    key,
		dir:    dir,
		cfg: Config{
			IdentityFile:   sshtest.WriteKeyFile(t, dir, "id_ed25519", key, ""),
			KnownHostsFile: sshtest.WriteKnownHosts(t, dir, srv.KnownHostsLine()),
			ConnectTimeout: 5 * time.Second,
		},
	}
}

// endpoint returns the server's endpoint.
func (e *testEnv) endpoint(t *testing.T) Endpoint {
	host, portStr, _ := net.SplitHostPort(e.server.Addr)
	port, _ := strconv.Atoi(portStr)
	return Endpoint{Name: "test", Host: host, Port: port, User: "tester"}
}

func (e *testEnv) dialer(t *testing.T) *Dialer {
	t.Helper()
	auth, err := NewAuth(e.cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { auth.Close() })
	d, err := NewDialer(e.cfg, auth)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDialSuccess(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{})
	c, err := env.dialer(t).Dial(context.Background(), env.endpoint(t))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	out, err := mustSession(t, c).Output("echo ok")
	if err != nil || string(out) != "ok\n" {
		t.Errorf("output = %q, %v", out, err)
	}
}

func mustSession(t *testing.T, c *ssh.Client) *ssh.Session {
	t.Helper()
	s, err := c.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestDialErrorKinds(t *testing.T) {
	t.Run("tcp refused", func(t *testing.T) {
		env := newTestEnv(t, sshtest.Options{})
		ln, _ := net.Listen("tcp", "127.0.0.1:0")
		addr := ln.Addr().(*net.TCPAddr)
		ln.Close() // nothing listens on this port now
		ep := Endpoint{Name: "gone", Host: "127.0.0.1", Port: addr.Port, User: "u"}
		_, err := env.dialer(t).Dial(context.Background(), ep)
		assertKind(t, err, errs.KindTCP)
	})
	t.Run("auth", func(t *testing.T) {
		env := newTestEnv(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{sshtest.NewKey(t).Public()}})
		_, err := env.dialer(t).Dial(context.Background(), env.endpoint(t))
		assertKind(t, err, errs.KindAuth)
	})
	t.Run("unknown host key", func(t *testing.T) {
		env := newTestEnv(t, sshtest.Options{})
		env.cfg.KnownHostsFile = sshtest.WriteKnownHosts(t, t.TempDir(), "")
		_, err := env.dialer(t).Dial(context.Background(), env.endpoint(t))
		assertKind(t, err, errs.KindHostKey)
		var hk *HostKeyError
		if !errors.As(err, &hk) || !hk.Unknown {
			t.Errorf("err = %v, want unknown host key", err)
		}
	})
	t.Run("changed host key", func(t *testing.T) {
		env := newTestEnv(t, sshtest.Options{})
		env.cfg.KnownHostsFile = sshtest.WriteKnownHosts(t, t.TempDir(),
			sshtest.KnownHostsLine(env.server.Addr, sshtest.NewKey(t).Public()))
		_, err := env.dialer(t).Dial(context.Background(), env.endpoint(t))
		var hk *HostKeyError
		if !errors.As(err, &hk) || hk.Unknown {
			t.Errorf("err = %v, want host key mismatch", err)
		}
	})
	t.Run("not ssh", func(t *testing.T) {
		env := newTestEnv(t, sshtest.Options{})
		ln, _ := net.Listen("tcp", "127.0.0.1:0")
		defer ln.Close()
		go func() {
			c, err := ln.Accept()
			if err == nil {
				c.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
				c.Close()
			}
		}()
		addr := ln.Addr().(*net.TCPAddr)
		_, err := env.dialer(t).Dial(context.Background(), Endpoint{Name: "web", Host: "127.0.0.1", Port: addr.Port, User: "u"})
		assertKind(t, err, errs.KindHandshake)
	})
	t.Run("canceled", func(t *testing.T) {
		env := newTestEnv(t, sshtest.Options{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := env.dialer(t).Dial(ctx, env.endpoint(t))
		assertKind(t, err, errs.KindCanceled)
	})
}

func TestClassifyDNS(t *testing.T) {
	err := classifyDial(context.Background(), Endpoint{Name: "x"}, &net.DNSError{Err: "no such host", Name: "x.invalid"})
	assertKind(t, err, errs.KindDNS)
}

func assertKind(t *testing.T, err error, want errs.Kind) {
	t.Helper()
	if got := errs.KindOf(err); got != want {
		t.Errorf("kind = %v (%v), want %v", got, err, want)
	}
}
