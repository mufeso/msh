package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHumanHandler(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, FormatText, true)
	log.Debug("loading inventory", "path", "/etc/msh/inventory.yaml")
	log.With("server", "web01").Warn("slow response")
	want := "msh: loading inventory path=/etc/msh/inventory.yaml\nmsh: warn: slow response server=web01\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestLevels(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, FormatText, false).Debug("hidden")
	if buf.Len() != 0 {
		t.Errorf("debug message without --verbose: %q", buf.String())
	}
}

func TestJSONHandler(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, FormatJSON, true).Debug("result", "server", "db01", "status", "ok")
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("not JSON: %q", buf.String())
	}
	if m["msg"] != "result" || m["server"] != "db01" || m["level"] != "DEBUG" {
		t.Errorf("record = %v", m)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Error("JSON log lines must end with a newline")
	}
}
