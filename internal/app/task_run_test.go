package app

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
)

// fleetConfig writes a config file with tasks for a fleet.
func fleetConfig(t *testing.T, tasks string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("tasks:\n"+tasks), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const basicTasks = `  hello:
    description: Say hello
    aliases: [hi]
    argv: [echo, hello world]
  pipeline:
    interpreter: sh
    command: "printf 'b\na\n' | sort"
  report:
    interpreter: sh
    script: |
      echo "line one"
      echo "line two"
  reboot:
    argv: [echo, would reboot]
    safety: dangerous
  greet:
    argv: [printf, "%s|%s\n", "{{1}}", "{{who}}"]
    arguments:
      - name: greeting
      - name: who
  grep:
    interpreter: sh
    command: "printf 'alpha\nbeta\ngamma beta\n' | grep -- {{pattern}}"
    arguments:
      - name: pattern
  args:
    interpreter: sh
    script: |
      printf '%s\n' "$1" "$2"
    arguments:
      - name: first
      - name: second
`

func TestTaskRunModes(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := fleetConfig(t, basicTasks)
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"task", "run", "hello", "--host", "s1"}, "hello world\n"},
		{[]string{"task", "run", "hi", "--host", "s1"}, "hello world\n"},
		{[]string{"task", "run", "pipeline", "--host", "s1"}, "a\nb\n"},
		{[]string{"task", "run", "report", "--host", "s1"}, "line one\nline two\n"},
		{[]string{"c", "hello", "s1"}, "hello world\n"},
	}
	for _, tt := range tests {
		stdout, stderr, code := fleet.run(t, tt.args, "--config", cfg)
		if exitcode.Code(code) != exitcode.OK {
			t.Errorf("%v: code = %d, stderr = %s", tt.args, code, stderr)
			continue
		}
		if !contains(stdout, tt.want) {
			t.Errorf("%v: stdout = %q, want %q", tt.args, stdout, tt.want)
		}
	}
}

func TestTaskRunRequiresYesForDangerous(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := fleetConfig(t, basicTasks)
	_, stderr, code := fleet.run(t, []string{"task", "run", "reboot", "--host", "s1"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, "--yes") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
	if len(fleet.servers[0].Commands()) != 0 {
		t.Error("dangerous task ran without --yes")
	}
	if _, _, code := fleet.run(t, []string{"task", "run", "reboot", "--host", "s1", "--yes"}, "--config", cfg); exitcode.Code(code) != exitcode.OK {
		t.Errorf("with --yes: code = %d", code)
	}
}

func TestTaskRunArguments(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := fleetConfig(t, basicTasks)
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"task", "run", "greet", "--host", "s1", "--arg", "hello, there", "--arg", "world"}, "hello, there|world\n"},
		{[]string{"task", "run", "grep", "--host", "s1", "--arg", "gamma beta"}, "gamma beta\n"},
		{[]string{"task", "run", "args", "--host", "s1", "--arg", "a b", "--arg", "$HOME"}, "a b\n$HOME\n"},
		{[]string{"c", "greet", "s1", "hi", "you"}, "hi|you\n"},
	}
	for _, tt := range tests {
		stdout, stderr, code := fleet.run(t, tt.args, "--config", cfg)
		if exitcode.Code(code) != exitcode.OK {
			t.Errorf("%v: code = %d, stderr = %s", tt.args, code, stderr)
			continue
		}
		if !contains(stdout, tt.want) {
			t.Errorf("%v: stdout = %q, want %q", tt.args, stdout, tt.want)
		}
	}
}

func TestTaskRunTooManyArguments(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := fleetConfig(t, basicTasks)
	_, stderr, code := fleet.run(t, []string{"c", "hello", "s1", "extra"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, "takes 0 argument(s)") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}
