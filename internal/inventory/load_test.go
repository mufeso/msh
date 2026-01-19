package inventory

import (
	"errors"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"example.com/msh/internal/decode"
)

func TestLoadFile(t *testing.T) {
	inv, err := LoadFile("testdata/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Servers) != 3 {
		t.Fatalf("got %d servers, want 3", len(inv.Servers))
	}
	if u := inv.Defaults.User; u == nil || *u != "admin" {
		t.Errorf("Defaults.User = %v", u)
	}
	web02, _ := inv.Find("web02")
	if web02.Port != 2222 || !web02.InGroup("staging") {
		t.Errorf("web02 = %+v", web02)
	}
	web01, _ := inv.Find("web01")
	if web01.Port != 0 {
		t.Errorf("LoadFile must not apply defaults; web01.Port = %d", web01.Port)
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

func TestLoadYAMLMatchesJSON(t *testing.T) {
	fromJSON, err := LoadFile("testdata/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := LoadFile("testdata/inventory.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromJSON, fromYAML) {
		t.Errorf("YAML and JSON inventories differ:\n%+v\n%+v", fromJSON, fromYAML)
	}
}

func TestDefaultsAcceptConfigKeys(t *testing.T) {
	in := "defaults:\n  workers: 3\n  connect_timeout: 4s\nservers:\n  - name: a\n    address: 192.0.2.1\n"
	inv, err := ParseFormat(strings.NewReader(in), decode.YAML)
	if err != nil {
		t.Fatal(err)
	}
	if w := inv.Defaults.Workers; w == nil || *w != 3 {
		t.Errorf("workers = %v", w)
	}
}

func TestDefaultsCannotSetInventory(t *testing.T) {
	in := "defaults:\n  inventory: other.yaml\nservers:\n  - name: a\n    address: 192.0.2.1\n"
	inv, err := ParseFormat(strings.NewReader(in), decode.YAML)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.Validate(); err == nil {
		t.Error("Validate accepted defaults.inventory")
	}
}

func TestLoadBastions(t *testing.T) {
	inv, err := LoadFile("testdata/bastion.yaml")
	if err != nil {
		t.Fatal(err)
	}
	b, ok := inv.FindBastion("production-bastion")
	if !ok || b.User != "jumpuser" || b.Address != "bastion.example.com" {
		t.Errorf("bastion = %+v, %v", b, ok)
	}
	app01, _ := inv.Find("app01")
	if app01.Bastion != "production-bastion" {
		t.Errorf("app01.Bastion = %q", app01.Bastion)
	}
	if _, ok := inv.FindBastion("nope"); ok {
		t.Error("FindBastion(nope) found something")
	}
}
