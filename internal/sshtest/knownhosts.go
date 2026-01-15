package sshtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// KnownHostsLine returns a known_hosts line for addr ("host:port").
func KnownHostsLine(addr string, key ssh.PublicKey) string {
	return knownhosts.Line([]string{knownhosts.Normalize(addr)}, key)
}

// WriteKnownHosts writes the given lines to dir/known_hosts and returns
// its path.
func WriteKnownHosts(t testing.TB, dir string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
