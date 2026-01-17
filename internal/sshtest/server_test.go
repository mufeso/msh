package sshtest

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// dial connects with plain x/crypto/ssh, verifying the host key.
func dial(t *testing.T, s *Server, key Key) *ssh.Client {
	t.Helper()
	known := WriteKnownHosts(t, t.TempDir(), s.KnownHostsLine())
	cb, err := knownhosts.New(known)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ssh.Dial("tcp", s.Addr, &ssh.ClientConfig{
		User:            "tester",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key.Signer)},
		HostKeyCallback: cb,
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestServerShellHandler(t *testing.T) {
	key := NewKey(t)
	s := NewServer(t, Options{AuthorizedKeys: []ssh.PublicKey{key.Public()}})
	sess, err := dial(t, s, key).NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	var out bytes.Buffer
	sess.Stdout = &out
	err = sess.Run("echo hello; exit 3")
	var exitErr *ssh.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitStatus() != 3 {
		t.Fatalf("err = %v, want exit status 3", err)
	}
	if out.String() != "hello\n" {
		t.Errorf("stdout = %q", out.String())
	}
	if got := s.Commands(); len(got) != 1 || got[0] != "echo hello; exit 3" {
		t.Errorf("commands = %q", got)
	}
}

func TestServerRejectsUnknownKey(t *testing.T) {
	s := NewServer(t, Options{AuthorizedKeys: []ssh.PublicKey{NewKey(t).Public()}})
	_, err := ssh.Dial("tcp", s.Addr, &ssh.ClientConfig{
		User:            "tester",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(NewKey(t).Signer)},
		HostKeyCallback: ssh.FixedHostKey(s.HostKey.Public()),
	})
	if err == nil {
		t.Fatal("unauthorized key accepted")
	}
}

func TestServerSignalStopsHandler(t *testing.T) {
	key := NewKey(t)
	s := NewServer(t, Options{AuthorizedKeys: []ssh.PublicKey{key.Public()}, Handler: Blocking("started\n")})
	sess, err := dial(t, s, key).NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.Start("wait"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := sess.Signal(ssh.SIGTERM); err != nil {
		t.Fatal(err)
	}
	_ = sess.Wait()
	if got := s.Signals(); len(got) != 1 || got[0] != "TERM" {
		t.Errorf("signals = %q", got)
	}
	if !s.WaitIdle(2 * time.Second) {
		t.Error("session still open on the server")
	}
}
