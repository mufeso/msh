package app

import (
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/sshtest"
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

func TestExecBastionFailureMessages(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	// Forget the bastion's host key: the bastion must fail, the target
	// must not be contacted directly.
	known := sshtest.WriteKnownHosts(t, t.TempDir(), fleet.servers[0].KnownHostsLine())
	stdout, _, code := fleet.run(t, []string{"exec"}, "--host", "s1", "--command", "true", "--known-hosts", known)
	if exitcode.Code(code) != exitcode.Connection {
		t.Errorf("code = %d, want connection", code)
	}
	if !contains(stdout, "bastion jump: host key verification failed") {
		t.Errorf("stdout = %q", stdout)
	}
	if len(fleet.servers[0].Commands()) != 0 {
		t.Error("target was contacted despite the bastion failure")
	}
}

func TestExecUnknownBastionFlag(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	_, stderr, code := fleet.run(t, []string{"exec"}, "--host", "s1", "--command", "true", "--bastion", "ghost")
	if exitcode.Code(code) != exitcode.Usage || !contains(stderr, `unknown bastion "ghost"`) {
		t.Errorf("code = %d, stderr = %q", code, stderr)
	}
}
