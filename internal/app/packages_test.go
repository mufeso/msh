package app

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/pkgs"
	"example.com/msh/internal/sshtest"
)

const rpmPackages = "#manager=rpm\nnginx\t1:1.24.0-1.el9\tx86_64\nopenssl\t1:3.0.7-16.el9\tx86_64\nbash\t0:5.1.8-6.el9\tx86_64\n"
const dpkgPackages = "#manager=dpkg\nnginx\t1.24.0-2ubuntu7\tamd64\nopenssl\t3.0.13-0ubuntu3\tamd64\ncurl\t8.5.0-2ubuntu10\tamd64\n"

// packageHandler answers the package script with output.
func packageHandler(output string) sshtest.Handler {
	return func(ctx context.Context, e *sshtest.Exec) int {
		if e.Command == pkgs.Command {
			io.WriteString(e.Stdout, output)
			return 0
		}
		return sshtest.ShellHandler(ctx, e)
	}
}

// packageFleet has s1 with rpm and s2 with dpkg.
func packageFleet(t *testing.T) *testFleet {
	t.Helper()
	return newFleet(t, fleetOptions{servers: 2, handlers: map[int]sshtest.Handler{1: packageHandler(rpmPackages), 2: packageHandler(dpkgPackages)}})
}

func TestPackagesList(t *testing.T) {
	fleet := packageFleet(t)
	stdout, stderr, code := fleet.run(t, []string{"packages", "list"}, "--all", "--filter", "open*", "--output", "csv")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "s1,openssl,1:3.0.7-16.el9,x86_64") || !contains(stdout, "s2,openssl,3.0.13-0ubuntu3,amd64") || contains(stdout, "nginx") {
		t.Errorf("stdout = %q", stdout)
	}
	if _, _, code := fleet.run(t, []string{"packages", "list"}, "--all", "--filter", "["); exitcode.Code(code) != exitcode.Usage {
		t.Errorf("bad pattern: code = %d", code)
	}
}

func TestPackagesNoManager(t *testing.T) {
	fleet := newTestFleet(t, 1, packageHandler("#manager=none\n"))
	_, stderr, code := fleet.run(t, []string{"packages", "list"}, "--all")
	if exitcode.Code(code) != exitcode.OK || !contains(stderr, "neither rpm nor dpkg") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}

func TestPackagesCheck(t *testing.T) {
	fleet := packageFleet(t)
	stdout, stderr, code := fleet.run(t, []string{"packages", "check"}, "--all", "--package", "nginx,curl")
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"s1", "curl", "missing", "1.24.0-2ubuntu7"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	if _, _, code := fleet.run(t, []string{"packages", "check"}, "--all", "--package", "nginx"); exitcode.Code(code) != exitcode.OK {
		t.Errorf("all present: code = %d", code)
	}
	if _, _, code := fleet.run(t, []string{"packages", "check"}, "--all"); exitcode.Code(code) != exitcode.Usage {
		t.Errorf("no package: code = %d", code)
	}
}

func TestPackagesCheckVersionsAcrossManagers(t *testing.T) {
	fleet := packageFleet(t)
	stdout, _, code := fleet.run(t, []string{"packages", "check"}, "--all", "--package", "openssl>=3.0.10")
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d", code)
	}
	if !contains(stdout, "wrong-version") || !contains(stdout, "3.0.13-0ubuntu3") {
		t.Errorf("stdout = %q", stdout)
	}
	if _, _, code := fleet.run(t, []string{"packages", "check"}, "--all", "--package", "openssl>=3.0,!telnet"); exitcode.Code(code) != exitcode.OK {
		t.Errorf("code = %d", code)
	}
}

func TestPackagesFind(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 3, handlers: map[int]sshtest.Handler{
		1: packageHandler(rpmPackages), 2: packageHandler(dpkgPackages), 3: packageHandler(rpmPackages),
	}})
	stdout, stderr, code := fleet.run(t, []string{"packages", "find", "open*"}, "--all", "--output", "csv")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "openssl,1:3.0.7-16.el9,2,\"s1,s3\"") || !contains(stdout, "openssl,3.0.13-0ubuntu3,1,s2") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestPackagesCompare(t *testing.T) {
	other := "#manager=rpm\nnginx\t1:1.26.0-1.el9\tx86_64\nopenssl\t1:3.0.7-16.el9\tx86_64\nextra\t0:1-1\tx86_64\n"
	fleet := newFleet(t, fleetOptions{servers: 3, handlers: map[int]sshtest.Handler{
		1: packageHandler(rpmPackages), 2: packageHandler(other), 3: packageHandler(rpmPackages),
	}})
	stdout, stderr, code := fleet.run(t, []string{"packages", "compare"}, "--all", "--output", "csv")
	if exitcode.Code(code) != exitcode.Failure {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	for _, want := range []string{"s2,bash,5.1.8-6.el9,-", "s2,extra,-,1-1", "s2,nginx,1:1.24.0-1.el9,1:1.26.0-1.el9"} {
		if !contains(stdout, want) {
			t.Errorf("missing %q in:\n%s", want, stdout)
		}
	}
	if contains(stdout, "s3,") {
		t.Errorf("identical servers must not appear:\n%s", stdout)
	}
	if _, stderr, code := fleet.run(t, []string{"packages", "compare"}, "--all", "--baseline", "ghost"); exitcode.Code(code) != exitcode.Usage || !contains(stderr, "baseline") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}

func TestPackagesCompareMixedManagersWarns(t *testing.T) {
	fleet := packageFleet(t)
	_, stderr, _ := fleet.run(t, []string{"packages", "compare"}, "--all")
	if !contains(stderr, "versions will not match") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestPackagesUpdates(t *testing.T) {
	dnf := sshtest.Static("#tool=dnf\nnginx.x86_64  1:1.26.0-1.el9  appstream\n", "", 0)
	apt := sshtest.Static("#tool=apt\nListing...\ncurl/noble-updates 8.5.0-2ubuntu10.1 amd64 [upgradable from: 8.5.0-2ubuntu10]\n", "", 0)
	fleet := newFleet(t, fleetOptions{servers: 2, handlers: map[int]sshtest.Handler{1: dnf, 2: apt}})
	stdout, stderr, code := fleet.run(t, []string{"packages", "updates"}, "--all", "--output", "csv")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "s1,nginx,1:1.26.0-1.el9,appstream") || !contains(stdout, "s2,curl,8.5.0-2ubuntu10.1,noble-updates") {
		t.Errorf("stdout = %q", stdout)
	}
	if cmds := fleet.servers[0].Commands(); cmds[0] != pkgs.UpdatesCommand {
		t.Errorf("commands = %q", cmds)
	}
}

func TestPackagesBaselineFile(t *testing.T) {
	fleet := packageFleet(t)
	stdout, stderr, code := fleet.run(t, []string{"packages", "check"}, "--all", "--only-problems", "--file", filepath.Join("..", "..", "examples", "packages-baseline.txt"))
	if exitcode.Code(code) != exitcode.Failure {
		t.Errorf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "s2") || !contains(stdout, "bash") || !contains(stdout, "missing") || contains(stdout, "ok") {
		t.Errorf("stdout = %q", stdout)
	}
	if _, _, code := fleet.run(t, []string{"packages", "check"}, "--all", "--file", "testdata/none.txt"); exitcode.Code(code) != exitcode.IO {
		t.Errorf("missing file: code = %d", code)
	}
}

// packageFactHandler answers both scripts.
func packageFactHandler(factsOut, packagesOut string) sshtest.Handler {
	f, p := factHandler(factsOut), packageHandler(packagesOut)
	return func(ctx context.Context, e *sshtest.Exec) int {
		if e.Command == pkgs.Command {
			return p(ctx, e)
		}
		return f(ctx, e)
	}
}

func TestPackagesWithWhereThroughBastion(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 2, bastion: true, handlers: map[int]sshtest.Handler{
		1: packageFactHandler(rockyFacts, rpmPackages), 2: packageFactHandler(ubuntuFacts, dpkgPackages),
	}})
	stdout, stderr, code := fleet.run(t, []string{"packages", "list"}, "--where", `package_manager == "dpkg"`, "--filter", "curl", "--output", "csv")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "s2,curl") || contains(stdout, "s1,") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestRealPackageScript(t *testing.T) {
	if _, err := exec.LookPath("rpm"); err != nil {
		if _, err := exec.LookPath("dpkg-query"); err != nil {
			t.Skip("neither rpm nor dpkg-query on this machine")
		}
	}
	fleet := newTestFleet(t, 1, nil)
	_, stderr, code := fleet.run(t, []string{"packages", "list"}, "--all")
	if exitcode.Code(code) != exitcode.OK {
		t.Errorf("code = %d, stderr = %s", code, stderr)
	}
}

func TestPackagePatternsWorkLikeGrep(t *testing.T) {
	fleet := packageFleet(t)
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"ssl"}, true},
		{[]string{"open"}, true},
		{[]string{"^open"}, true},
		{[]string{"^ssl"}, false},
		{[]string{"*ssl*"}, true},
		{[]string{"SSL"}, false},
		{[]string{"SSL", "--ignore-case"}, true},
		{[]string{"--filter", "ssl"}, true},
	}
	for _, tt := range tests {
		for _, cmd := range [][]string{{"packages", "list"}, {"packages", "find"}} {
			if cmd[1] == "find" && tt.args[0] == "--filter" {
				continue
			}
			stdout, stderr, code := fleet.run(t, cmd, append([]string{"--all"}, tt.args...)...)
			if exitcode.Code(code) != exitcode.OK {
				t.Fatalf("%v %v: code = %d, stderr = %s", cmd, tt.args, code, stderr)
			}
			if contains(stdout, "openssl") != tt.want || contains(stdout, "nginx") {
				t.Errorf("%v %v: stdout = %q", cmd, tt.args, stdout)
			}
		}
	}
	if _, stderr, code := fleet.run(t, []string{"packages", "find"}, "--all", "("); exitcode.Code(code) != exitcode.Usage || !contains(stderr, "invalid pattern") {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}
