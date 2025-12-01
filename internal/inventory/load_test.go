package inventory

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

func TestLoadFile(t *testing.T) {
	inv, err := LoadFile("testdata/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Servers) != 3 {
		t.Fatalf("got %d servers, want 3", len(inv.Servers))
	}
	if inv.Defaults.User != "admin" {
		t.Errorf("Defaults.User = %q", inv.Defaults.User)
	}
	web02, _ := inv.Find("web02")
	if web02.Port != 2222 || !web02.InGroup("staging") {
		t.Errorf("web02 = %+v", web02)
	}
}

func TestLoadFileMissing(t *testing.T) {
	_, err := LoadFile("testdata/does-not-exist.json")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want fs.ErrNotExist", err)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	_, err := Parse(strings.NewReader(`{"servers":[{"name":"a","adress":"x"}]}`))
	if err == nil || !strings.Contains(err.Error(), "adress") {
		t.Errorf("err = %v, want unknown field error", err)
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	if _, err := Parse(strings.NewReader(`{"servers": [`)); err == nil {
		t.Error("Parse(truncated) = nil error")
	}
}
