package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

// historyFiles returns the records under home.
func historyFiles(t *testing.T, home string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(home, ".local", "state", "msh", "history", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRunsAreRecorded(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	home := t.TempDir()
	env := map[string]string{"HOME": home}
	args := append([]string{"exec", "--all", "--command", "echo hi"}, fleet.args()...)
	if _, stderr, code := runCLIEnv(t, env, args...); code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	files := historyFiles(t, home)
	if len(files) != 1 {
		t.Fatalf("files = %v", files)
	}
	info, _ := os.Stat(files[0])
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", info.Mode().Perm())
	}

	runCLIEnv(t, env, append(args, "--no-history")...)
	env["MSH_HISTORY"] = "false"
	runCLIEnv(t, env, args...)
	if n := len(historyFiles(t, home)); n != 1 {
		t.Errorf("%d records, want 1", n)
	}
}

func TestHistoryFailureDoesNotChangeTheResult(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"XDG_STATE_HOME": blocked} // a file, not a directory
	args := append([]string{"exec", "--all", "--command", "true"}, fleet.args()...)
	_, stderr, code := runCLIEnv(t, env, args...)
	if code != exitcode.OK || !contains(stderr, "warning: could not save") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

// fleetHistory runs a few commands and returns the environment.
func fleetHistory(t *testing.T, fleet *testFleet, commands ...string) map[string]string {
	t.Helper()
	env := map[string]string{"HOME": t.TempDir()}
	for _, cmd := range commands {
		args := append([]string{"exec", "--all", "--command", cmd}, fleet.args()...)
		runCLIEnv(t, env, args...)
	}
	return env
}

func TestHistoryList(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	env := fleetHistory(t, fleet, "echo one", "exit 3")
	stdout, stderr, code := runCLIEnv(t, env, "history", "list")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	for _, want := range []string{"PROBLEMS", "'echo one'", "'exit 3'"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	stdout, _, _ = runCLIEnv(t, env, "history", "list", "--limit", "1", "--output", "jsonl")
	if n := strings.Count(stdout, "\n"); n != 1 || !contains(stdout, `"problems":2`) {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestHistoryShow(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	env := fleetHistory(t, fleet, "echo remembered")
	stdout, stderr, code := runCLIEnv(t, env, "history", "show", "last")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	for _, want := range []string{"Command: msh exec", "=== s1: ok", "remembered\n"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	if _, stderr, code := runCLIEnv(t, env, "history", "show", "1999"); code != exitcode.Usage || !contains(stderr, "no such run") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func historyIDs(t *testing.T, env map[string]string) []string {
	t.Helper()
	stdout, _, _ := runCLIEnv(t, env, "history", "list", "--output", "json")
	var rows []historyRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestHistoryDiff(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	env := fleetHistory(t, fleet, "echo before", "echo after; exit 2")
	ids := historyIDs(t, env)
	if len(ids) != 2 {
		t.Fatalf("ids = %v", ids)
	}
	stdout, stderr, code := runCLIEnv(t, env, "history", "diff", ids[1], ids[0])
	if code != exitcode.Failure {
		t.Errorf("code = %v, stderr = %s", code, stderr)
	}
	for _, want := range []string{"status", "failed", "exit_code", "stdout", "before", "after"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	if _, _, code := runCLIEnv(t, env, "history", "diff", ids[0], ids[0]); code != exitcode.OK {
		t.Errorf("same run: code = %v", code)
	}
}

func TestHistoryLimit(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	home := t.TempDir()
	env := map[string]string{"HOME": home, "MSH_HISTORY_LIMIT": "2"}
	args := append([]string{"exec", "--all", "--command", "true"}, fleet.args()...)
	for range 4 {
		runCLIEnv(t, env, args...)
	}
	if n := len(historyFiles(t, home)); n != 2 {
		t.Errorf("%d records, want 2", n)
	}
	stdout, _, code := runCLIEnv(t, env, "history", "prune", "--keep", "1")
	if code != exitcode.OK || !contains(stdout, "removed 1 runs") {
		t.Errorf("code = %v, stdout = %q", code, stdout)
	}
}

func TestHistoryFilters(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	env := fleetHistory(t, fleet, "echo fine", "exit 3")
	rows := func(args ...string) string {
		stdout, stderr, code := runCLIEnv(t, env, append([]string{"history", "list"}, args...)...)
		if code != exitcode.OK {
			t.Fatalf("code = %v, stderr = %s", code, stderr)
		}
		return stdout
	}
	if out := rows("--failed"); !contains(out, "exit 3") || contains(out, "echo fine") {
		t.Errorf("--failed:\n%s", out)
	}
	if out := rows("--grep", "fine"); !contains(out, "echo fine") || contains(out, "exit 3") {
		t.Errorf("--grep:\n%s", out)
	}
	if out := rows("--server", "ghost"); contains(out, "echo") {
		t.Errorf("--server:\n%s", out)
	}
	if out := rows("--since", "1h"); !contains(out, "echo fine") {
		t.Errorf("--since:\n%s", out)
	}
}
