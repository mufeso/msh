package config

import (
	"strings"
	"testing"
	"time"

	"example.com/msh/internal/decode"
)

func TestDurationInYAMLAndJSON(t *testing.T) {
	inputs := map[decode.Format]string{
		decode.YAML: "connect_timeout: 5s\ncommand_timeout: 2m30s\n",
		decode.JSON: `{"connect_timeout": "5s", "command_timeout": "2m30s"}`,
	}
	for format, in := range inputs {
		l, err := Parse(strings.NewReader(in), format)
		if err != nil {
			t.Fatalf("format %d: %v", format, err)
		}
		if l.ConnectTimeout == nil || time.Duration(*l.ConnectTimeout) != 5*time.Second {
			t.Errorf("format %d: connect_timeout = %v", format, l.ConnectTimeout)
		}
		if l.CommandTimeout == nil || time.Duration(*l.CommandTimeout) != 150*time.Second {
			t.Errorf("format %d: command_timeout = %v", format, l.CommandTimeout)
		}
	}
}

func TestDurationRejectsGarbage(t *testing.T) {
	_, err := Parse(strings.NewReader("connect_timeout: soon\n"), decode.YAML)
	if err == nil || !strings.Contains(err.Error(), "invalid duration") {
		t.Errorf("err = %v", err)
	}
}

func TestDurationMarshalText(t *testing.T) {
	b, _ := Duration(90 * time.Second).MarshalText()
	if string(b) != "1m30s" {
		t.Errorf("MarshalText = %q", b)
	}
}
