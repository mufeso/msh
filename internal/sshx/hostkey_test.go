package sshx

import (
	"errors"
	"net"
	"slices"
	"strings"
	"testing"

	"example.com/msh/internal/sshtest"
)

func TestHostKeyCallback(t *testing.T) {
	dir := t.TempDir()
	known := sshtest.NewKey(t)
	other := sshtest.NewKey(t)
	addr := "192.0.2.10:22"
	file := sshtest.WriteKnownHosts(t, dir, sshtest.KnownHostsLine(addr, known.Public()))

	cb, err := HostKeyCallback(Config{KnownHostsFile: file})
	if err != nil {
		t.Fatal(err)
	}
	remote := &net.TCPAddr{IP: net.ParseIP("192.0.2.10"), Port: 22}

	if err := cb(addr, remote, known.Public()); err != nil {
		t.Errorf("known key rejected: %v", err)
	}

	var hkErr *HostKeyError
	err = cb(addr, remote, other.Public())
	if !errors.As(err, &hkErr) || hkErr.Unknown {
		t.Fatalf("changed key: err = %v, want mismatch HostKeyError", err)
	}
	if !strings.Contains(err.Error(), "MISMATCH") {
		t.Errorf("message = %q", err)
	}

	err = cb("192.0.2.99:22", &net.TCPAddr{IP: net.ParseIP("192.0.2.99"), Port: 22}, other.Public())
	if !errors.As(err, &hkErr) || !hkErr.Unknown {
		t.Fatalf("unknown host: err = %v, want unknown HostKeyError", err)
	}
}

func TestHostKeyCallbackRequiresFile(t *testing.T) {
	if _, err := HostKeyCallback(Config{}); err == nil {
		t.Error("empty known_hosts path accepted")
	}
	if _, err := HostKeyCallback(Config{KnownHostsFile: t.TempDir() + "/missing"}); err == nil {
		t.Error("missing known_hosts file accepted")
	}
}

func TestInsecureMustBeExplicit(t *testing.T) {
	cb, err := HostKeyCallback(Config{InsecureIgnoreHostKey: true})
	if err != nil || cb == nil {
		t.Fatalf("insecure callback: %v", err)
	}
	if err := cb("x:22", &net.TCPAddr{}, sshtest.NewKey(t).Public()); err != nil {
		t.Errorf("insecure callback rejected a key: %v", err)
	}
}

func TestHostKeyAlgorithms(t *testing.T) {
	dir := t.TempDir()
	addr := "192.0.2.10:2222"
	file := sshtest.WriteKnownHosts(t, dir, sshtest.KnownHostsLine(addr, sshtest.NewKey(t).Public()))
	got := HostKeyAlgorithms(Config{KnownHostsFile: file}, addr)
	if !slices.Equal(got, []string{"ssh-ed25519"}) {
		t.Errorf("algorithms = %v", got)
	}
	if got := HostKeyAlgorithms(Config{KnownHostsFile: file}, "192.0.2.11:22"); got != nil {
		t.Errorf("unknown host algorithms = %v, want nil", got)
	}
}
