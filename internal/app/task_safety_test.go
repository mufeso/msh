package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

// hostileValues are arguments an attacker (or a typo) might pass. Each
// must reach the remote program as one unchanged argument.
func hostileValues(marker string) []string {
	return []string{
		"two words",
		"it's",
		`say "hi"`,
		"$(touch " + marker + ")",
		"`touch " + marker + "`",
		"x; touch " + marker,
		"x | touch " + marker,
		"x && touch " + marker,
		"*",
		"~",
		"$HOME",
		"line1\nline2",
		"-n",
		"--help",
		"'; touch " + marker + "; '",
		`\`,
		"",
	}
}

const safetyTasks = `  direct:
    argv: [printf, "[%s]", "{{v}}"]
    arguments:
      - name: v
  shell:
    interpreter: sh
    command: "printf '[%s]' {{v}} | cat"
    arguments:
      - name: v
  bashshell:
    interpreter: bash
    command: "printf '[%s]' {{v}}"
    arguments:
      - name: v
  script:
    interpreter: sh
    script: |
      printf '[%s]' "$1"
    arguments:
      - name: v
`

func TestTaskArgumentsCannotInjectShellSyntax(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := fleetConfig(t, safetyTasks)
	marker := filepath.Join(t.TempDir(), "pwned")

	for _, taskName := range []string{"direct", "shell", "bashshell", "script"} {
		for _, v := range hostileValues(marker) {
			stdout, stderr, code := fleet.run(t, []string{"task", "run", taskName, "--host", "s1", "--arg", v}, "--config", cfg, "--output", "json")
			if exitcode.Code(code) != exitcode.OK {
				t.Errorf("%s %q: code = %d, stderr = %s", taskName, v, code, stderr)
				continue
			}
			var views []output.ResultView
			if err := json.Unmarshal([]byte(stdout), &views); err != nil || len(views) != 1 {
				t.Fatalf("%s %q: bad JSON: %v", taskName, v, err)
			}
			if want := "[" + v + "]"; views[0].Stdout != want {
				t.Errorf("%s: value changed: sent %q, received %q", taskName, v, views[0].Stdout)
			}
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("%s: an argument executed a command (marker file exists)", taskName)
		}
	}
}
