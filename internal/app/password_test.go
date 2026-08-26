package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/sshtest"
)

const testPassword = "correct-horse-test"

// passwordServer starts a server that knows no key and returns it with a
// known_hosts file that trusts it.
func passwordServer(t *testing.T, opts sshtest.Options) (*sshtest.Server, string) {
	t.Helper()
	opts.Password = testPassword
	srv := sshtest.NewServer(t, opts)
	known := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(known, []byte(srv.KnownHostsLine()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return srv, known
}

func TestPasswordFromEnvironment(t *testing.T) {
	srv, known := passwordServer(t, sshtest.Options{})
	env := map[string]string{"MSH_PASSWORD": testPassword}
	stdout, stderr, code := runCLICtx(t, context.Background(), env, "--verbose", "--known-hosts", known, "tester@"+srv.Addr, "echo", "ok")
	if code != exitcode.OK || stdout != "ok\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if contains(stdout+stderr, testPassword) {
		t.Error("the password was printed")
	}
}

func TestWrongPasswordIsNotPrinted(t *testing.T) {
	srv, known := passwordServer(t, sshtest.Options{})
	env := map[string]string{"MSH_PASSWORD": "wrong-test-value"}
	stdout, stderr, code := runCLICtx(t, context.Background(), env, "--known-hosts", known, "tester@"+srv.Addr, "true")
	if code != exitcode.Connection {
		t.Errorf("code = %v, stderr = %s", code, stderr)
	}
	if contains(stdout+stderr, "wrong-test-value") {
		t.Error("the password was printed")
	}
}

func TestKeysComeBeforeThePassword(t *testing.T) {
	key := sshtest.NewKey(t)
	srv, known := passwordServer(t, sshtest.Options{AuthorizedKeys: nil})
	_ = key
	_, _, code := runCLICtx(t, context.Background(), nil, "--known-hosts", known, "tester@"+srv.Addr, "true")
	if code != exitcode.Connection {
		t.Errorf("no key and no password: code = %v", code)
	}
}
