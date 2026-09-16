package app

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/facts"
	"example.com/msh/internal/sshtest"
)

const rockyFacts = "os_id=rocky\nos_like=rhel fedora\nos_version=9.4\nkernel=5.14\narch=x86_64\ncpus=8\nmemory_mb=16000\ndisk_root_pct=91\npackage_manager=rpm\n"
const ubuntuFacts = "os_id=ubuntu\nos_like=debian\nos_version=24.04\nkernel=6.8\narch=aarch64\ncpus=2\nmemory_mb=4000\ndisk_root_pct=40\npackage_manager=dpkg\n"

// factHandler answers the facts script with output and runs everything
// else in a real shell.
func factHandler(output string) sshtest.Handler {
	return func(ctx context.Context, e *sshtest.Exec) int {
		if e.Command == facts.Command {
			io.WriteString(e.Stdout, output)
			return 0
		}
		return sshtest.ShellHandler(ctx, e)
	}
}

// mixedFleet has s1 on Rocky and s2 on Ubuntu.
func mixedFleet(t *testing.T) *testFleet {
	t.Helper()
	return newFleet(t, fleetOptions{servers: 2, handlers: map[int]sshtest.Handler{1: factHandler(rockyFacts), 2: factHandler(ubuntuFacts)}})
}

func TestFactsCommand(t *testing.T) {
	fleet := mixedFleet(t)
	stdout, stderr, code := fleet.run(t, []string{"facts"}, "--all", "--output", "json")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	var views []factsView
	if err := json.Unmarshal([]byte(stdout), &views); err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].Facts["os_family"] != "redhat" || views[1].Facts["os_family"] != "debian" {
		t.Errorf("views = %+v", views)
	}
	stdout, _, _ = fleet.run(t, []string{"facts"}, "--all")
	if !contains(stdout, "rocky") || !contains(stdout, "dpkg") {
		t.Errorf("table = %q", stdout)
	}
}

func TestFactsReportsUnreachableServers(t *testing.T) {
	fleet := mixedFleet(t)
	fleet.servers[1].Close()
	stdout, stderr, code := fleet.run(t, []string{"facts"}, "--all")
	if exitcode.Code(code) != exitcode.Connection || !contains(stderr, "s2") || !contains(stdout, "rocky") {
		t.Errorf("code = %d, stderr = %q, stdout = %q", code, stderr, stdout)
	}
}

func TestFactsCacheAndRefresh(t *testing.T) {
	fleet := mixedFleet(t)
	env := map[string]string{"HOME": t.TempDir()}
	args := append([]string{"facts", "--all"}, fleet.args()...)
	runCLIEnv(t, env, args...)
	runCLIEnv(t, env, args...)
	if n := len(fleet.servers[0].Commands()); n != 1 {
		t.Errorf("%d collections, want 1", n)
	}
	runCLIEnv(t, env, append(args, "--refresh")...)
	if n := len(fleet.servers[0].Commands()); n != 2 {
		t.Errorf("%d collections after --refresh, want 2", n)
	}
}

func TestWhere(t *testing.T) {
	fleet := mixedFleet(t)
	stdout, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--where", `os_family == "redhat"`, "--command", "echo picked")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "=== s1: ok") || contains(stdout, "=== s2") {
		t.Errorf("stdout = %q", stdout)
	}
	stdout, _, _ = fleet.run(t, []string{"exec"}, "--where", `cpus < 4 && name ~ "^s"`, "--command", "echo picked")
	if !contains(stdout, "=== s2: ok") || contains(stdout, "=== s1") {
		t.Errorf("--where alone: %q", stdout)
	}
}

func TestWhereErrors(t *testing.T) {
	fleet := mixedFleet(t)
	if _, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--where", "a ==", "--command", "true"); exitcode.Code(code) != exitcode.Usage || !contains(stderr, "--where") {
		t.Errorf("bad expression: code = %d, stderr = %q", code, stderr)
	}
	if _, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--where", `os_id == "nope"`, "--command", "true"); exitcode.Code(code) != exitcode.Usage || !contains(stderr, "no servers match") {
		t.Errorf("no match: code = %d, stderr = %q", code, stderr)
	}
	fleet.servers[1].Close()
	stdout, stderr, _ := fleet.run(t, []string{"exec"}, "--all", "--where", "cpus > 0", "--command", "true")
	if !contains(stderr, "warning: s2") || contains(stdout, "=== s2") {
		t.Errorf("unreachable: stdout = %q, stderr = %q", stdout, stderr)
	}
}

func TestRunbookWhen(t *testing.T) {
	fleet := mixedFleet(t)
	cfg := writeConfig(t, `runbooks:
  update:
    steps:
      - name: redhat
        when: os_family == "redhat"
        command: "echo dnf"
        read_only: true
      - name: debian
        when: os_family == "debian"
        command: "echo apt"
        read_only: true
`)
	stdout, stderr, code := fleet.run(t, []string{"runbook", "run", "update", "--all", "--ordered", "--prefix"}, "--config", cfg)
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"s1_: dnf", "s2_: apt", "s1_: --- step debian: skipped", "s2_: --- step redhat: skipped"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	dry, _, _ := fleet.run(t, []string{"runbook", "run", "update", "--all", "--dry-run"}, "--config", cfg)
	if !contains(dry, `when: os_family == "redhat"`) {
		t.Errorf("dry run = %q", dry)
	}
}

func TestFactsExpect(t *testing.T) {
	fleet := mixedFleet(t)
	stdout, _, code := fleet.run(t, []string{"facts"}, "--all", "--expect", "disk_root_pct < 90", "--expect", "cpus >= 2")
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d", code)
	}
	if !contains(stdout, "s1") || !contains(stdout, "disk_root_pct < 90") || contains(stdout, "s2") {
		t.Errorf("stdout = %q", stdout)
	}
	if _, _, code := fleet.run(t, []string{"facts"}, "--all", "--expect", "cpus >= 2"); exitcode.Code(code) != exitcode.OK {
		t.Errorf("all good: code = %d", code)
	}
	if _, _, code := fleet.run(t, []string{"facts"}, "--all", "--expect", "cpus >="); exitcode.Code(code) != exitcode.Usage {
		t.Errorf("bad expression: code = %d", code)
	}
}

func TestFactsGroupBy(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 3, handlers: map[int]sshtest.Handler{
		1: factHandler(rockyFacts), 2: factHandler(ubuntuFacts), 3: factHandler(rockyFacts),
	}})
	stdout, _, code := fleet.run(t, []string{"facts"}, "--all", "--group-by", "os_family", "--output", "csv")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d", code)
	}
	if !contains(stdout, "OS_FAMILY,COUNT,SERVERS\nredhat,2,\"s1,s3\"\ndebian,1,s2\n") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestWhereUsesInventoryNames(t *testing.T) {
	fleet := mixedFleet(t)
	stdout, stderr, code := fleet.run(t, []string{"exec"}, "--all", "--where", `name == "s2" && user == "tester" && facts.cpus == 2`, "--command", "echo ok")
	if exitcode.Code(code) != exitcode.OK || !contains(stdout, "=== s2: ok") {
		t.Errorf("code = %d, stdout = %q, stderr = %q", code, stdout, stderr)
	}
}
