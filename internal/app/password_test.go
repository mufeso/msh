package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
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

func TestKeyboardInteractivePassword(t *testing.T) {
	srv, known := passwordServer(t, sshtest.Options{KeyboardOnly: true})
	env := map[string]string{"MSH_PASSWORD": testPassword}
	stdout, stderr, code := runCLICtx(t, context.Background(), env, "--known-hosts", known, "tester@"+srv.Addr, "echo", "ok")
	if code != exitcode.OK || stdout != "ok\n" || srv.PasswordTries() != 1 {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

// fakeReader answers prompts and records them.
type fakeReader struct {
	mu      sync.Mutex
	prompts []string
	delay   time.Duration
}

func (r *fakeReader) read(_ context.Context, prompt string) (string, error) {
	time.Sleep(r.delay)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prompts = append(r.prompts, prompt)
	return testPassword, nil
}

func (r *fakeReader) configure(c *cli) {
	c.passwordReader = func() (interactive.ReadPassword, bool) { return r.read, true }
}

func TestPasswordPrompt(t *testing.T) {
	srv, known := passwordServer(t, sshtest.Options{})
	r := &fakeReader{}
	stdout, stderr, code := runCLIWith(t, context.Background(), nil, r.configure, "--known-hosts", known, "tester@"+srv.Addr, "echo", "ok")
	if code != exitcode.OK || stdout != "ok\n" {
		t.Fatalf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if len(r.prompts) != 1 || r.prompts[0] != "tester@127.0.0.1's password: " {
		t.Errorf("prompts = %q", r.prompts)
	}
}

func TestOnePromptForAFleet(t *testing.T) {
	a, knownA := passwordServer(t, sshtest.Options{})
	b, knownB := passwordServer(t, sshtest.Options{})
	lines, _ := os.ReadFile(knownB)
	f, err := os.OpenFile(knownA, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(lines)
	f.Close()
	r := &fakeReader{}
	stdout, stderr, code := runCLIWith(t, context.Background(), nil, r.configure, "--known-hosts", knownA, "--host", "tester@"+a.Addr, "tester@"+b.Addr, "echo", "ok")
	if code != exitcode.OK || !contains(stdout, "--- 2 servers: 2 ok") {
		t.Fatalf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if len(r.prompts) != 1 {
		t.Errorf("prompts = %q", r.prompts)
	}
}

func TestTypingDoesNotCountAsConnectTime(t *testing.T) {
	srv, known := passwordServer(t, sshtest.Options{})
	r := &fakeReader{delay: 300 * time.Millisecond}
	stdout, stderr, code := runCLIWith(t, context.Background(), nil, r.configure, "--connect-timeout", "100ms", "--known-hosts", known, "tester@"+srv.Addr, "echo", "ok")
	if code != exitcode.OK || stdout != "ok\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestNoTerminalMeansNoPrompt(t *testing.T) {
	srv, known := passwordServer(t, sshtest.Options{})
	configure := func(c *cli) {
		c.passwordReader = func() (interactive.ReadPassword, bool) { return nil, false }
	}
	_, stderr, code := runCLIWith(t, context.Background(), nil, configure, "--known-hosts", known, "tester@"+srv.Addr, "true")
	if code != exitcode.Connection || !contains(stderr, "no usable SSH keys") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}
