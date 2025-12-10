package config

import (
	"strings"
	"testing"

	"example.com/msh/internal/decode"
)

func TestApplyOnlyOverridesSetFields(t *testing.T) {
	s := Defaults()
	user := "ops"
	s.Apply(Layer{User: &user})
	if s.User != "ops" {
		t.Errorf("User = %q, want ops", s.User)
	}
	if s.Port != 22 || s.Output != "text" {
		t.Errorf("unset fields changed: %+v", s)
	}
}

func TestApplyCanSetZeroValue(t *testing.T) {
	// A pointer to "" is different from nil: it explicitly clears the value.
	s := Defaults()
	empty := ""
	s.Apply(Layer{Output: &empty})
	if s.Output != "" {
		t.Errorf("Output = %q, want empty", s.Output)
	}
}

func TestLoadFile(t *testing.T) {
	l, err := LoadFile("testdata/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if l.User == nil || *l.User != "ops" || l.Port == nil || *l.Port != 2222 {
		t.Errorf("layer = %+v", l)
	}
	if l.Output != nil {
		t.Errorf("Output should be unset, got %q", *l.Output)
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	_, err := Parse(strings.NewReader(`{"usr": "x"}`), decode.JSON)
	if err == nil || !strings.Contains(err.Error(), "usr") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadYAMLFile(t *testing.T) {
	l, err := LoadFile("testdata/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if l.User == nil || *l.User != "ops" || l.Output == nil || *l.Output != "json" {
		t.Errorf("layer = %+v", l)
	}
}
