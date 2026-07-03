package app

import (
	"context"
	"io"
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
