package cliflag

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStringListRepeatedAndComma(t *testing.T) {
	var hosts StringList
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(&hosts, "host", "hosts")
	if err := fs.Parse([]string{"--host", "a", "--host", "b, c"}); err != nil {
		t.Fatal(err)
	}
	if want := (StringList{"a", "b", "c"}); !reflect.DeepEqual(hosts, want) {
		t.Errorf("hosts = %v, want %v", hosts, want)
	}
	if got := hosts.String(); got != "a,b,c" {
		t.Errorf("String() = %q", got)
	}
}

func TestStringListRejectsEmptyItems(t *testing.T) {
	var l StringList
	if err := l.Set("a,,b"); err == nil {
		t.Error("Set(a,,b) = nil, want error")
	}
}

func TestKeyValues(t *testing.T) {
	var tags KeyValues
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&tags, "tag", "tags")
	if err := fs.Parse([]string{"--tag", "env=prod", "--tag", "role = web"}); err != nil {
		t.Fatal(err)
	}
	want := KeyValues{"env": "prod", "role": "web"}
	if !reflect.DeepEqual(tags, want) {
		t.Errorf("tags = %v, want %v", tags, want)
	}
	if got := tags.String(); got != "env=prod,role=web" {
		t.Errorf("String() = %q", got)
	}
	if err := fs.Parse([]string{"--tag", "novalue"}); err == nil {
		t.Error("parsing --tag novalue succeeded, want error")
	}
}

func TestHostList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.txt")
	content := "# web tier\nweb01 web02\nroot@db01:2222, db02  # primary\n\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	var hosts HostList
	for _, v := range []string{"a,b", "c d", "  e\tf ", "user@g", "@" + path} {
		if err := hosts.Set(v); err != nil {
			t.Fatalf("Set(%q): %v", v, err)
		}
	}
	want := HostList{"a", "b", "c", "d", "e", "f", "user@g", "web01", "web02", "root@db01:2222", "db02"}
	if !reflect.DeepEqual(hosts, want) {
		t.Errorf("hosts = %v", hosts)
	}
}

func TestHostListErrors(t *testing.T) {
	var hosts HostList
	if err := hosts.Set(" , "); err == nil {
		t.Error("empty list accepted")
	}
	if err := hosts.Set("@/does/not/exist"); err == nil {
		t.Error("missing file accepted")
	}
	empty := filepath.Join(t.TempDir(), "empty")
	os.WriteFile(empty, []byte("# nothing\n"), 0o600)
	if err := hosts.ReadFile(empty); err == nil {
		t.Error("empty file accepted")
	}
}
