package cliflag

import (
	"flag"
	"io"
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
