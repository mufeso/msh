package app

import (
	"os/exec"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

func completions(t *testing.T, words ...string) string {
	t.Helper()
	stdout, stderr, code := runCLI(t, append([]string{"__complete"}, words...)...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	return stdout
}

func TestCompleteCommands(t *testing.T) {
	if got := completions(t, "pa"); got != "packages\n" {
		t.Errorf("got %q", got)
	}
	if got := completions(t, "packages", "c"); got != "check\ncompare\n" {
		t.Errorf("got %q", got)
	}
	if got := completions(t, "__"); got != "" {
		t.Errorf("hidden command completed: %q", got)
	}
}

func TestEverySubcommandTableEntryWorks(t *testing.T) {
	for _, cmd := range commands() {
		for _, sub := range subcommands(cmd.name) {
			if cmd.name == "completion" || cmd.name == "script" {
				continue
			}
			// "--help" is answered by every subcommand's flag set; an
			// unknown subcommand would report an error instead.
			_, stderr, _ := runCLI(t, cmd.name, sub, "--help")
			if strings.Contains(stderr, "unknown") {
				t.Errorf("%s %s: %s", cmd.name, sub, stderr)
			}
		}
	}
}

func TestBashScriptSyntax(t *testing.T) {
	stdout, _, code := runCLI(t, "completion", "bash")
	if code != exitcode.OK {
		t.Fatalf("code = %v", code)
	}
	cmd := exec.Command("bash", "-n")
	cmd.Stdin = strings.NewReader(stdout)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("bash -n: %v\n%s", err, out)
	}
	if _, _, code := runCLI(t, "completion", "tcsh"); code != exitcode.Usage {
		t.Errorf("code = %v", code)
	}
}

func TestDynamicCompletion(t *testing.T) {
	cfg := writeConfig(t, "inventory: testdata/inventory.yaml\n"+basicRunbooks)
	env := map[string]string{"MSH_CONFIG": cfg}
	got := func(words ...string) string {
		stdout, _, _ := runCLIEnv(t, env, append([]string{"__complete"}, words...)...)
		return stdout
	}
	if s := got("exec", "--host", "we"); s != "web01\nweb02\n" {
		t.Errorf("--host: %q", s)
	}
	if s := got("exec", "--group", "d"); s != "database\n" {
		t.Errorf("--group: %q", s)
	}
	if s := got("runbook", "run", "gr"); s != "greet\n" {
		t.Errorf("runbook: %q", s)
	}
	if s := got("c", "he"); s != "hello\n" {
		t.Errorf("c task: %q", s)
	}
	if s := got("c", "hello", "db"); s != "db01\n" {
		t.Errorf("c server: %q", s)
	}
	if s := got("exec", "--wh"); s != "--where\n" {
		t.Errorf("flags: %q", s)
	}
}

func TestCompletionIsSilentOnErrors(t *testing.T) {
	stdout, stderr, code := runCLI(t, "__complete", "exec", "--host", "x")
	if code != exitcode.OK || stdout != "" || stderr != "" {
		t.Errorf("code = %v, stdout = %q, stderr = %q", code, stdout, stderr)
	}
}
