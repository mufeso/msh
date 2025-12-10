package inventory

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"
)

func TestLoadFile(t *testing.T) {
	inv, err := LoadFile("testdata/inventory.json", "local")
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Servers) != 3 {
		t.Fatalf("got %d servers, want 3", len(inv.Servers))
	}
	if inv.Defaults.User != "admin" {
		t.Errorf("Defaults.User = %q", inv.Defaults.User)
	}
	web01, _ := inv.Find("web01")
	if web01.Port != 22 {
		t.Errorf("web01.Port = %d, want default 22", web01.Port)
	}
	if web01.User != "admin" {
		t.Errorf("web01.User = %q, want inventory default admin", web01.User)
	}
	db01, _ := inv.Find("db01")
	if db01.User != "dbadmin" {
		t.Errorf("db01.User = %q, want dbadmin", db01.User)
	}
	web02, _ := inv.Find("web02")
	if web02.Port != 2222 || !web02.InGroup("staging") {
		t.Errorf("web02 = %+v", web02)
	}
}

func TestLoadFileMissing(t *testing.T) {
	_, err := LoadFile("testdata/does-not-exist.json", "local")
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

func TestLoadYAMLMatchesJSON(t *testing.T) {
	fromJSON, err := LoadFile("testdata/inventory.json", "local")
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := LoadFile("testdata/inventory.yaml", "local")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromJSON, fromYAML) {
		t.Errorf("YAML and JSON inventories differ:\n%+v\n%+v", fromJSON, fromYAML)
	}
}
