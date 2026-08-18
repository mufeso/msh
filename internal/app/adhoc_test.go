package app

import (
	"context"
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

func TestConnectLikeSSH(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	ctx := withInput(context.Background(), "echo in-the-shell\nexit 5\n")
	args := append(fleet.credentials(), "tester@"+fleet.servers[0].Addr)
	stdout, stderr, code := runCLICtx(t, ctx, nil, args...)
	if code != 5 || !contains(stdout, "in-the-shell") {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestConnectByInventoryName(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	ctx := withInput(context.Background(), "exit 6\n")
	cfg := writeConfig(t, "inventory: "+fleet.inventory+"\n")
	stdout, stderr, code := runCLICtx(t, ctx, map[string]string{"MSH_CONFIG": cfg}, append(fleet.credentials(), "s1")...)
	if code != 6 {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestTypoIsNotADestination(t *testing.T) {
	_, stderr, code := runCLI(t, "verison")
	if code != exitcode.Usage || !contains(stderr, `Did you mean "version"?`) || !contains(stderr, "msh connect verison") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}

func TestOptionLikeDestinationIsRejected(t *testing.T) {
	_, stderr, code := runCLI(t, "connect", "--", "-oProxyCommand=x")
	if code != exitcode.Usage || !contains(stderr, "invalid host") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}
