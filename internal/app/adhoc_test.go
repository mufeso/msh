package app

import (
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
)

// credentials are the flags to reach a fleet without its inventory.
func (f *testFleet) credentials() []string {
	return []string{"--identity", f.identity, "--known-hosts", f.known}
}

func TestExecWithoutInventory(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	hosts := "tester@" + fleet.servers[0].Addr + " tester@" + fleet.servers[1].Addr
	args := append([]string{"exec", "--host", hosts, "--command", "echo adhoc"}, fleet.credentials()...)
	stdout, stderr, code := runCLI(t, args...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if strings.Count(stdout, ": ok") != 2 || !contains(stdout, "adhoc") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestHostKeysAreStillVerifiedForAdHocServers(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	other := newTestFleet(t, 1, nil)
	args := []string{"exec", "--host", "tester@" + fleet.servers[0].Addr, "--command", "true",
		"--identity", fleet.identity, "--known-hosts", other.known}
	stdout, _, code := runCLI(t, args...)
	if code != exitcode.Connection || !contains(stdout, "host key verification failed") {
		t.Errorf("code = %v, stdout = %q", code, stdout)
	}
	if len(fleet.servers[0].Commands()) != 0 {
		t.Error("a command ran on an unverified server")
	}
}

func TestAdHocBastion(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	args := append([]string{"exec", "--host", "tester@" + fleet.servers[0].Addr, "--bastion", "tester@" + fleet.bastion.Addr, "--command", "echo jumped"}, fleet.credentials()...)
	stdout, stderr, code := runCLI(t, args...)
	if code != exitcode.OK || !contains(stdout, "jumped") {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if len(fleet.bastion.Forwards()) != 1 {
		t.Errorf("forwards = %v", fleet.bastion.Forwards())
	}
}

func TestGroupStillNeedsAnInventory(t *testing.T) {
	_, stderr, code := runCLI(t, "exec", "--group", "web", "--command", "true")
	if code != exitcode.Config || !contains(stderr, "no inventory file") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}
