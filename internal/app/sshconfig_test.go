package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"example.com/msh/internal/exitcode"
)

// sshHome returns a HOME with the given ~/.ssh/config.
func sshHome(t *testing.T, content string) map[string]string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return map[string]string{"HOME": home}
}

func hostBlock(t *testing.T, alias, addr, extra string) string {
	t.Helper()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("Host %s\n  HostName %s\n  Port %s\n  User tester\n%s\n", alias, host, port, extra)
}

func TestAliasFromSSHConfig(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	env := sshHome(t, hostBlock(t, "myserver", fleet.servers[0].Addr, "  IdentityFile "+fleet.identity))
	// No --identity: the key must come from the file.
	stdout, stderr, code := runCLICtx(t, context.Background(), env, "--known-hosts", fleet.known, "myserver", "echo", "alias")
	if code != exitcode.OK || stdout != "alias\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestDestinationBeatsSSHConfig(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	_, port, _ := net.SplitHostPort(fleet.servers[0].Addr)
	env := sshHome(t, "Host myserver\n  HostName 127.0.0.1\n  Port 1\n  User wrong\n")
	args := append(fleet.credentials(), "tester@myserver:"+port, "echo", "ok")
	stdout, stderr, code := runCLICtx(t, context.Background(), env, args...)
	if code != exitcode.OK || stdout != "ok\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestProxyJumpFromSSHConfig(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	env := sshHome(t, hostBlock(t, "jump", fleet.bastion.Addr, "")+
		hostBlock(t, "target", fleet.servers[0].Addr, "  ProxyJump jump"))
	args := append(fleet.credentials(), "target", "echo", "jumped")
	stdout, stderr, code := runCLICtx(t, context.Background(), env, args...)
	if code != exitcode.OK || stdout != "jumped\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if len(fleet.bastion.Forwards()) != 1 {
		t.Errorf("forwards = %v", fleet.bastion.Forwards())
	}
}

func TestJumpFlagTakesAnAlias(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	env := sshHome(t, hostBlock(t, "jump", fleet.bastion.Addr, ""))
	args := append(fleet.credentials(), "-J", "jump", "tester@"+fleet.servers[0].Addr, "echo", "ok")
	stdout, stderr, code := runCLICtx(t, context.Background(), env, args...)
	if code != exitcode.OK || stdout != "ok\n" || len(fleet.bastion.Forwards()) != 1 {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestSeveralHopsNeverConnectDirectly(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	env := sshHome(t, hostBlock(t, "target", fleet.servers[0].Addr, "  ProxyJump a.example.com,b.example.com"))
	args := append(fleet.credentials(), "target", "true")
	_, stderr, code := runCLICtx(t, context.Background(), env, args...)
	if code != exitcode.Usage || !contains(stderr, "several hops") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
	if len(fleet.servers[0].Commands()) != 0 {
		t.Error("connected directly")
	}
}

func TestBrokenSSHConfigIsOnlyAWarning(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	env := sshHome(t, "Port nonsense\n")
	args := append(fleet.credentials(), "tester@"+fleet.servers[0].Addr, "echo", "ok")
	stdout, stderr, code := runCLICtx(t, context.Background(), env, args...)
	if code != exitcode.OK || stdout != "ok\n" || !contains(stderr, "ignoring SSH configuration") {
		t.Errorf("code = %v, stdout = %q, stderr = %q", code, stdout, stderr)
	}
}

func TestInventoryServersIgnoreSSHConfig(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	env := sshHome(t, "Host s1\n  HostName 192.0.2.1\n  Port 1\n")
	args := append([]string{"exec", "--host", "s1", "--command", "true"}, fleet.args()...)
	if _, stderr, code := runCLICtx(t, context.Background(), env, args...); code != exitcode.OK {
		t.Errorf("code = %v, stderr = %s", code, stderr)
	}
}
