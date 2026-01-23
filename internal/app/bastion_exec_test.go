package app

import (
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestExecThroughBastion(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	stdout, stderr, code := fleet.run(t, []string{"exec"}, "--host", "s1", "--command", "echo via-jump")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if !contains(stdout, "via-jump") {
		t.Errorf("stdout = %q", stdout)
	}
	if fw := fleet.bastion.Forwards(); len(fw) != 1 || fw[0] != fleet.servers[0].Addr {
		t.Errorf("bastion forwards = %v", fw)
	}
}

func TestExecBastionNoneConnectsDirectly(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	_, stderr, code := fleet.run(t, []string{"exec"}, "--host", "s1", "--command", "true", "--bastion", "none")
	if exitcode.Code(code) != exitcode.OK {
		t.Fatalf("code = %d, stderr = %s", code, stderr)
	}
	if fw := fleet.bastion.Forwards(); len(fw) != 0 {
		t.Errorf("bastion was used despite --bastion none: %v", fw)
	}
}
