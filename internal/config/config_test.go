package config

import (
	"strings"
	"testing"

	"example.com/msh/internal/decode"
)

func ptr[T any](v T) *T { return &v }

func TestResolvePrecedence(t *testing.T) {
	file := Layer{User: ptr("fileuser"), Port: ptr(2222)}
	flags := Layer{User: ptr("flaguser")}

	s := Resolve(Source{SourceFile, file}, Source{SourceFlag, flags})

	if s.User != "flaguser" || s.Sources[KeyUser] != SourceFlag {
		t.Errorf("user = %q from %q, want flaguser from flag", s.User, s.Sources[KeyUser])
	}
	if s.Port != 2222 || s.Sources[KeyPort] != SourceFile {
		t.Errorf("port = %d from %q, want 2222 from config file", s.Port, s.Sources[KeyPort])
	}
	if s.Output != "text" || s.Sources[KeyOutput] != SourceDefault {
		t.Errorf("output = %q from %q, want text from default", s.Output, s.Sources[KeyOutput])
	}
}

func TestApplyCanSetZeroValue(t *testing.T) {
	s := Resolve(Source{SourceFlag, Layer{User: ptr("")}})
	if s.User != "" || s.Sources[KeyUser] != SourceFlag {
		t.Errorf("explicit empty value not applied: %+v", s)
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
