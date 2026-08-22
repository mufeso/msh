package sshconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sample = `# global
IdentityFile ~/.ssh/global

Host web? !web9
    HostName %h.example.com
    User=deploy
    Port 2222
    ProxyJump "jump.example.com"   # trailing comment

Host db
    HOSTNAME 192.0.2.20
    User first
    User second
    IdentityFile ~/.ssh/db

Match exec "something"
    User never

Host *
    User fallback
    ServerAliveInterval 30
`

func TestLookup(t *testing.T) {
	c, err := Parse(strings.NewReader(sample), "/home/me")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]Host{
		"web1": {HostName: "web1.example.com", User: "deploy", Port: 2222, ProxyJump: "jump.example.com",
			IdentityFiles: []string{"/home/me/.ssh/global"}},
		"web9": {User: "fallback", IdentityFiles: []string{"/home/me/.ssh/global"}},
		"db": {HostName: "192.0.2.20", User: "first",
			IdentityFiles: []string{"/home/me/.ssh/global", "/home/me/.ssh/db"}},
	}
	for alias, want := range tests {
		if got := c.Lookup(alias); !reflect.DeepEqual(got, want) {
			t.Errorf("Lookup(%q) = %+v", alias, got)
		}
	}
	if !c.Known("db") || c.Known("other") {
		t.Error("Known is wrong")
	}
}

func TestNilConfig(t *testing.T) {
	var c *Config
	if !reflect.DeepEqual(c.Lookup("x"), Host{}) || c.Known("x") {
		t.Error("a nil Config must be empty")
	}
}

func TestErrors(t *testing.T) {
	for _, bad := range []string{"Port abc\n", "Port 70000\n", "User\n"} {
		if _, err := Parse(strings.NewReader(bad), ""); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestInclude(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".ssh", "conf.d")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	main := filepath.Join(home, ".ssh", "config")
	write(main, "Include conf.d/*.conf\nHost a\n  User late\nHost b\n  Include ~/.ssh/inner\n")
	write(filepath.Join(dir, "one.conf"), "Host a\n  User early\n")
	write(filepath.Join(home, ".ssh", "inner"), "Port 2200\n")
	c, err := Load(main, home)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Lookup("a"); got.User != "early" || got.Port != 0 {
		t.Errorf("a = %+v", got)
	}
	if got := c.Lookup("b"); got.Port != 2200 {
		t.Errorf("an Include inside a Host block must stay inside it: %+v", got)
	}
}

func TestIncludeLoopStops(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(home, ".ssh", "config")
	if err := os.WriteFile(main, []byte("Include config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(main, home); err == nil || !strings.Contains(err.Error(), "too deeply") {
		t.Errorf("err = %v", err)
	}
}
