package sshtest

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh/agent"
)

// StartAgent runs an in-memory ssh-agent holding keys and returns its
// socket path. It stops when the test ends.
func StartAgent(t testing.TB, keys ...Key) string {
	t.Helper()
	keyring := agent.NewKeyring()
	for _, k := range keys {
		if err := keyring.Add(agent.AddedKey{PrivateKey: k.Private}); err != nil {
			t.Fatal(err)
		}
	}

	// Unix socket paths are limited to about 104 bytes on macOS, and
	// t.TempDir() paths can be longer, so use a short directory.
	dir, err := os.MkdirTemp("", "msh-agent")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "agent.sock")

	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					t.Logf("agent accept: %v", err)
				}
				return
			}
			go func() {
				defer conn.Close()
				_ = agent.ServeAgent(keyring, conn)
			}()
		}
	}()
	return socket
}
