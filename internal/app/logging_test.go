package app

import (
	"encoding/json"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestJSONRunEvents(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	_, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--verbose", "--log-format", "json", "--command", "echo SECRET-OUTPUT")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d", code)
	}
	var results, finished int
	for _, line := range strings.Split(strings.TrimSpace(stderr), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("stderr line is not JSON: %q", line)
		}
		switch m["msg"] {
		case "result":
			results++
			if m["status"] != "ok" || m["server"] == nil {
				t.Errorf("result event = %v", m)
			}
		case "run finished":
			finished++
		}
	}
	if results != 2 || finished != 1 {
		t.Errorf("results = %d, finished = %d\n%s", results, finished, stderr)
	}
	if strings.Contains(stderr, "SECRET-OUTPUT") {
		t.Error("command output leaked into the logs")
	}
}

func TestBadLogFormat(t *testing.T) {
	if _, _, code := runCLI(t, "config", "show", "--log-format", "xml"); code != exitcode.Usage {
		t.Errorf("code = %v", code)
	}
}
