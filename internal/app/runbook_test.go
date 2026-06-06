package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
	"example.com/msh/internal/sshtest"
)

// writeConfig writes a configuration file for a test.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInvalidRunbookIsConfigError(t *testing.T) {
	cfg := writeConfig(t, "runbooks:\n  broken:\n    steps:\n      - name: x\n        task: nope\n")
	_, stderr, code := runCLI(t, "config", "show", "--config", cfg)
	if code != exitcode.Config || !contains(stderr, `unknown task "nope"`) {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

const basicRunbooks = `tasks:
  hello:
    argv: [echo, hello]
runbooks:
  greet:
    description: Say hello twice
    steps:
      - name: first
        task: hello
      - name: second
        command: "echo second | cat"
        read_only: true
`

func TestRunbookList(t *testing.T) {
	stdout, stderr, code := runCLI(t, "runbook", "list", "--config", writeConfig(t, basicRunbooks))
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "greet") || !contains(stdout, "Say hello twice") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestRunbookShow(t *testing.T) {
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := runCLI(t, "runbook", "show", "greet", "--config", cfg)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	for _, want := range []string{"1. first", "task: hello", "2. second", "command: echo second | cat"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	_, stderr, code = runCLI(t, "runbook", "show", "gret", "--config", cfg)
	if code != exitcode.Usage || !contains(stderr, `Did you mean "greet"?`) {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func TestRunbookDryRunContactsNobody(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "greet", "--all", "--dry-run"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"2 steps on 2 servers", "Batch 1: s1, s2", "$ echo hello", "$ sh -c 'echo second | cat'"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	for _, s := range fleet.servers {
		if len(s.Commands()) != 0 {
			t.Error("--dry-run contacted a server")
		}
	}
}

func TestRunbookRun(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "greet", "--all"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"=== s1: ok", "--- step first: ok", "hello\n", "--- step second: ok", "second\n"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	if len(fleet.servers[1].Commands()) != 2 {
		t.Errorf("s2 commands = %q", fleet.servers[1].Commands())
	}
}

func TestRunbookStopsOnFailure(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := writeConfig(t, `runbooks:
  bad:
    steps:
      - name: fail
        command: "exit 3"
        read_only: true
      - name: never
        command: "echo never"
        read_only: true
`)
	stdout, _, code := fleet.run(t, []string{"runbook", "run", "bad", "--host", "s1"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d", code)
	}
	if contains(stdout, "never") || !contains(stdout, "--- step fail: failed (exit 3)") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestRunbookJSONSteps(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "greet", "--all", "--workers", "3", "--output", "json"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	var run output.RunView
	if err := json.Unmarshal([]byte(stdout), &run); err != nil {
		t.Fatal(err)
	}
	if len(run.Results) != 3 {
		t.Fatalf("results = %d", len(run.Results))
	}
	for _, r := range run.Results {
		if len(r.Steps) != 2 || r.Steps[0].Name != "first" || r.Steps[1].Stdout != "second\n" {
			t.Errorf("steps = %+v", r.Steps)
		}
	}
}

func TestRunbookTimeouts(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := writeConfig(t, `runbooks:
  step-limit:
    steps:
      - name: slow
        command: "sleep 5"
        read_only: true
        timeout: 200ms
      - name: after
        command: "echo after"
        read_only: true
  total-limit:
    timeout: 300ms
    steps:
      - name: one
        command: "sleep 5"
        read_only: true
      - name: two
        command: "echo two"
        read_only: true
`)
	for _, name := range []string{"step-limit", "total-limit"} {
		start := time.Now()
		stdout, _, code := fleet.run(t, []string{"runbook", "run", name, "--host", "s1"}, "--config", cfg)
		if exitcode.Code(code) != exitcode.Timeout {
			t.Errorf("%s: code = %d\n%s", name, code, stdout)
		}
		if time.Since(start) > 4*time.Second {
			t.Errorf("%s: timeout not enforced", name)
		}
		if contains(stdout, "after\n") || contains(stdout, "two\n") {
			t.Errorf("%s: a later step ran:\n%s", name, stdout)
		}
	}
}

func TestRunbookNeedsYes(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := writeConfig(t, `runbooks:
  change:
    steps:
      - name: touch
        command: "echo would change"
`)
	_, stderr, code := fleet.run(t, []string{"runbook", "run", "change", "--host", "s1"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, "--yes") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
	if len(fleet.servers[0].Commands()) != 0 {
		t.Error("a changing runbook ran without --yes")
	}
	if _, _, code := fleet.run(t, []string{"runbook", "run", "change", "--host", "s1", "--yes"}, "--config", cfg); exitcode.Code(code) != exitcode.OK {
		t.Errorf("with --yes: code = %d", code)
	}
	stdout, _, _ := fleet.run(t, []string{"runbook", "run", "change", "--host", "s1", "--dry-run"}, "--config", cfg)
	if !contains(stdout, "Safety:  changes") {
		t.Errorf("dry run = %q", stdout)
	}
}

const argumentRunbooks = `tasks:
  show:
    argv: [printf, "[%s]", "{{value}}"]
    arguments:
      - name: value
        required: true
runbooks:
  args:
    arguments:
      - name: who
        required: true
        pattern: "[a-z ;$()]+"
      - name: extra
        default: fallback
    steps:
      - name: via-task
        task: show
        params: {value: "hello {{who}}"}
      - name: via-command
        command: "printf '<%s>' {{who}} {{extra}}"
        read_only: true
`

func TestRunbookArguments(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	cfg := writeConfig(t, argumentRunbooks)
	marker := "a b; (x) $(id)"
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "args", "--host", "s1", "--param", "who=" + marker}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"[hello " + marker + "]", "<" + marker + ">", "<fallback>"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	_, stderr, code = fleet.run(t, []string{"runbook", "run", "args", "--host", "s1"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, "is required") {
		t.Errorf("missing argument: code = %d, stderr = %q", code, stderr)
	}
	_, stderr, code = fleet.run(t, []string{"runbook", "run", "args", "--host", "s1", "--arg", "BAD"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, "does not match") {
		t.Errorf("bad pattern: code = %d, stderr = %q", code, stderr)
	}
}

func TestRunbookUnknownPlaceholderIsConfigError(t *testing.T) {
	cfg := writeConfig(t, "tasks:\n  t:\n    argv: [echo, \"{{v}}\"]\n    arguments:\n      - name: v\nrunbooks:\n  r:\n    steps:\n      - name: s\n        task: t\n        params: {v: \"{{ghost}}\"}\n")
	if _, _, code := runCLI(t, "runbook", "list", "--config", cfg); code != exitcode.Config {
		t.Errorf("code = %v", code)
	}
}

func TestRunbookResume(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	dir := t.TempDir()
	count := filepath.Join(dir, "count")
	gate := filepath.Join(dir, "gate")
	cfg := writeConfig(t, `runbooks:
  r:
    arguments:
      - name: count
      - name: gate
    steps:
      - name: once
        command: "echo x >> {{count}}"
        allow_redirection: true
      - name: gated
        command: "test -f {{gate}}"
        read_only: true
      - name: last
        command: "echo finished"
        read_only: true
`)
	env := map[string]string{"HOME": t.TempDir()}
	args := append([]string{"runbook", "run", "r", "--all", "--yes", "--config", cfg, "--param", "count=" + count, "--param", "gate=" + gate}, fleet.args()...)

	_, stderr, code := runCLIEnv(t, env, args...)
	if code != exitcode.Failure || !contains(stderr, "--resume") {
		t.Fatalf("first run: code = %v, stderr = %q", code, stderr)
	}
	if err := os.WriteFile(gate, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runCLIEnv(t, env, append(args, "--resume")...)
	if code != exitcode.OK {
		t.Fatalf("resume: code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "--- step once: already done") || !contains(stdout, "finished") {
		t.Errorf("stdout = %q", stdout)
	}
	data, _ := os.ReadFile(count)
	if len(data) != 4 { // "x\n" once per server, not twice
		t.Errorf("the first step ran again: %q", data)
	}
	// Success removed the state, so there is nothing left to resume.
	_, stderr, code = runCLIEnv(t, env, append(args, "--resume")...)
	if code != exitcode.Usage || !contains(stderr, "no saved progress") {
		t.Errorf("after success: code = %v, stderr = %q", code, stderr)
	}
}

func TestRunbookEndToEndThroughBastion(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 3, bastion: true, handlers: map[int]sshtest.Handler{2: sshtest.Static("", "broken\n", 4)}})
	cfg := writeConfig(t, basicRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "greet", "--all", "--ordered", "--output", "json"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Failure {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	var run output.RunView
	if err := json.Unmarshal([]byte(stdout), &run); err != nil {
		t.Fatal(err)
	}
	if run.Results[0].Status != "ok" || run.Results[1].Status != "failed" || run.Results[2].Status != "ok" {
		t.Errorf("results = %+v", run.Results)
	}
	if s := run.Results[1].Steps; s[0].ExitCode != 4 || s[1].Status != "skipped" {
		t.Errorf("s2 steps = %+v", s)
	}
	if run.Summary.Failed != 1 || len(fleet.bastion.Forwards()) < 5 {
		t.Errorf("summary = %+v, forwards = %d", run.Summary, len(fleet.bastion.Forwards()))
	}
}

func TestExampleRunbookDryRun(t *testing.T) {
	stdout, stderr, code := runCLI(t, "runbook", "run", "restart-web", "--all", "--dry-run", "--config", exampleConfig, "--inventory", exampleInventory, "--param", "service=httpd")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "sudo systemctl restart httpd") || !contains(stdout, "is-active httpd") {
		t.Errorf("stdout = %q", stdout)
	}
}

const rollingRunbooks = `runbooks:
  roll:
    batch: 2
    steps:
      - name: work
        command: "echo working"
        read_only: true
`

func TestRollingRunbook(t *testing.T) {
	fleet := newTestFleet(t, 5, nil)
	cfg := writeConfig(t, rollingRunbooks)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "roll", "--all", "--prefix"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "--- 5 servers: 5 ok") {
		t.Errorf("one merged summary expected:\n%s", stdout)
	}
}

func TestRollingRunbookStopsOnFailure(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 5, handlers: map[int]sshtest.Handler{2: sshtest.Static("", "", 1)}})
	cfg := writeConfig(t, rollingRunbooks)
	stdout, _, code := fleet.run(t, []string{"runbook", "run", "roll", "--all"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d", code)
	}
	if !contains(stdout, "=== s3: skipped") || !contains(stdout, "rollout stopped: 1 failed servers (limit 0)") {
		t.Errorf("stdout = %q", stdout)
	}
	if len(fleet.servers[3].Commands()) != 0 {
		t.Error("a later batch ran")
	}
}

func TestRolloutFlags(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 4, handlers: map[int]sshtest.Handler{1: sshtest.Static("", "", 1)}})
	cfg := writeConfig(t, rollingRunbooks)
	stdout, _, _ := fleet.run(t, []string{"runbook", "run", "roll", "--all", "--batch", "25%", "--max-failures", "1"}, "--config", cfg)
	if contains(stdout, "skipped") {
		t.Errorf("one failure is within the limit:\n%s", stdout)
	}
	if _, stderr, code := fleet.run(t, []string{"runbook", "run", "roll", "--all", "--batch", "many"}, "--config", cfg); exitcode.Code(code) != exitcode.Usage || !contains(stderr, "batch:") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}

func TestDryRunShowsBatches(t *testing.T) {
	fleet := newTestFleet(t, 5, nil)
	cfg := writeConfig(t, rollingRunbooks)
	stdout, _, _ := fleet.run(t, []string{"runbook", "run", "roll", "--all", "--dry-run"}, "--config", cfg)
	for _, want := range []string{"Batch 1: s1, s2", "Batch 3: s5", "Stops after more than 0 failed servers"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
}

func TestRolloutApproval(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	cfg := writeConfig(t, rollingRunbooks)
	args := append([]string{"runbook", "run", "roll", "--all", "--batch", "1", "--approve", "--config", cfg}, fleet.args()...)

	ctx := withInput(context.Background(), "y\nn\n")
	stdout, stderr, code := runCLICtx(t, ctx, nil, args...)
	if code != exitcode.Failure {
		t.Errorf("code = %v", code)
	}
	if !contains(stderr, "Continue with batch 2/3 (s2)?") || !contains(stderr, "Continue with batch 3/3 (s3)?") {
		t.Errorf("stderr = %q", stderr)
	}
	if !contains(stdout, "=== s3: skipped") || !contains(stdout, "not approved") {
		t.Errorf("stdout = %q", stdout)
	}
	if len(fleet.servers[1].Commands()) != 1 || len(fleet.servers[2].Commands()) != 0 {
		t.Error("approval did not control the batches")
	}

	if _, stderr, code := runCLICtx(t, context.Background(), nil, append(args, "--auto-approve")...); code != exitcode.OK || contains(stderr, "Continue") {
		t.Errorf("--auto-approve: code = %v, stderr = %q", code, stderr)
	}
}

func TestApprovalNeedsTerminal(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	cfg := writeConfig(t, rollingRunbooks)
	configure := func(c *cli) { c.term.(*testTerminal).tty = false }
	args := append([]string{"runbook", "run", "roll", "--all", "--approve", "--config", cfg}, fleet.args()...)
	_, stderr, code := runCLIWith(t, context.Background(), nil, configure, args...)
	if code != exitcode.Usage || !contains(stderr, "--auto-approve") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}
