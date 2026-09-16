package app

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/sshtest"
	"example.com/msh/internal/tmux"
	"example.com/msh/internal/tmux/tmuxtest"
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
	if strings.Count(stdout, ": adhoc\n") != 2 || !contains(stdout, "2 ok") {
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

func TestRemoteCommandLikeSSH(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	dest := "tester@" + fleet.servers[0].Addr
	args := append(fleet.credentials(), dest, "echo", "raw", "output;", "exit", "4")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != 4 || stdout != "raw output\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if cmds := fleet.servers[0].Commands(); cmds[0] != "echo raw output; exit 4" {
		t.Errorf("remote command = %q", cmds)
	}
	if len(fleet.servers[0].PTYs()) != 0 {
		t.Error("a plain remote command must not use a terminal")
	}
}

func TestFlagsAroundDestinationAndTTY(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	dest := "tester@" + fleet.servers[0].Addr
	// -t after the destination; "-l" after the command word stays in it.
	args := append([]string{"connect", dest}, fleet.credentials()...)
	args = append(args, "-t", "ls", "-l", "/dev/null")
	_, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if cmds := fleet.servers[0].Commands(); cmds[0] != "ls -l /dev/null" {
		t.Errorf("remote command = %q", cmds)
	}
	if len(fleet.servers[0].PTYs()) != 1 {
		t.Error("-t did not request a terminal")
	}
}

func TestSSHFlags(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, bastion: true})
	host, port, _ := net.SplitHostPort(fleet.servers[0].Addr)
	args := []string{"-l", "tester", "-p", port, "-i", fleet.identity, "--known-hosts", fleet.known,
		"-J", "tester@" + fleet.bastion.Addr, host, "echo", "flags-work"}
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || stdout != "flags-work\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	if len(fleet.bastion.Forwards()) != 1 {
		t.Errorf("forwards = %v", fleet.bastion.Forwards())
	}
}

func TestPortFlagBeatsInventory(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	_, port, _ := net.SplitHostPort(fleet.servers[0].Addr)
	inv := filepath.Join(t.TempDir(), "inv.yaml")
	content := "servers:\n  - name: web\n    address: 127.0.0.1\n    port: 1\n    user: wrong\n"
	if err := os.WriteFile(inv, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	args := append(fleet.credentials(), "--inventory", inv, "-p", port, "-l", "tester", "web", "echo", "ok")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || stdout != "ok\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestSeveralHostsWithoutCommandUseTmux(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	hosts := "tester@" + fleet.servers[0].Addr + " tester@" + fleet.servers[1].Addr
	fake := tmuxtest.NewFake()
	configure := func(c *cli) {
		c.lookPath = func(string) (string, error) { return "/usr/bin/tmux", nil }
		c.newTmux = func(string) tmux.Runner { return fake }
	}
	args := append(fleet.credentials(), "--host", hosts, "--tmux-detached")
	_, stderr, code := runCLIWith(t, context.Background(), nil, configure, args...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	created := fake.CallsTo("new-session")
	if len(created) != 1 || len(fake.CallsTo("split-window")) != 1 {
		t.Fatalf("calls = %v", fake.Calls())
	}
	if line := created[0][len(created[0])-1]; contains(line, "--inventory") {
		t.Errorf("panes must not ask for an inventory file that does not exist:\n%s", line)
	}
}

func TestDestinationOverridesInventoryServer(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	_, port, _ := net.SplitHostPort(fleet.servers[0].Addr)
	inv := filepath.Join(t.TempDir(), "inv.yaml")
	content := "servers:\n  - name: web\n    address: 127.0.0.1\n    port: 1\n    user: wrong\n"
	if err := os.WriteFile(inv, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	args := append(fleet.credentials(), "--inventory", inv, "tester@web:"+port, "echo", "ok")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || stdout != "ok\n" {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestPanesKeepTheUser(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	hosts := fleet.servers[0].Addr + " " + fleet.servers[1].Addr
	fake := tmuxtest.NewFake()
	configure := func(c *cli) {
		c.lookPath = func(string) (string, error) { return "/usr/bin/tmux", nil }
		c.newTmux = func(string) tmux.Runner { return fake }
	}
	args := append(fleet.credentials(), "-l", "someone", "--host", hosts, "--tmux-detached")
	_, stderr, code := runCLIWith(t, context.Background(), nil, configure, args...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	created := fake.CallsTo("new-session")
	if line := created[0][len(created[0])-1]; !contains(line, "-l someone") {
		t.Errorf("pane lost the user:\n%s", line)
	}
}

func TestHostFlagAddsToTheDestination(t *testing.T) {
	fleet := newTestFleet(t, 3, nil)
	a, b := "tester@"+fleet.servers[0].Addr, "tester@"+fleet.servers[1].Addr
	dest := "tester@" + fleet.servers[2].Addr
	file := filepath.Join(t.TempDir(), "a")
	if err := os.WriteFile(file, []byte(a+"\n"+b+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, list := range []string{a + " " + b, a + "," + b, a + ", " + b, "@" + file} {
		args := append(fleet.credentials(), "--host", list, dest, "echo", "everywhere")
		stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
		if code != exitcode.OK {
			t.Fatalf("--host %q: code = %v, stderr = %s", list, code, stderr)
		}
		if strings.Count(stdout, "everywhere") != 3 || !contains(stdout, ": 3 servers: 3 ok") {
			t.Errorf("--host %q: stdout = %q", list, stdout)
		}
	}
	for _, s := range fleet.servers {
		if cmds := s.Commands(); cmds[0] != "echo everywhere" {
			t.Errorf("remote command = %q", cmds)
		}
	}
}

func TestWordsAfterAllAreTheCommand(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	args := append(fleet.args(), "--all", "echo", "all")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || strings.Count(stdout, "all\n") != 2 {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}

func TestParseRemote(t *testing.T) {
	tests := []struct {
		args      []string
		keep      int
		wantWords []string
		wantHosts []string
	}{
		{[]string{"tmux", "ops@web01"}, 1, []string{"tmux"}, []string{"ops@web01"}},
		{[]string{"ops@web01", "tmux"}, 1, []string{"tmux"}, []string{"ops@web01"}},
		{[]string{"web01", "web02"}, 0, nil, []string{"web01", "web02"}},
		{[]string{"tmux", "web01"}, 1, []string{"tmux"}, []string{"web01"}},
		{[]string{"tmux", "nginx>=1.20", "ops@web01"}, -1, []string{"tmux", "nginx>=1.20"}, []string{"ops@web01"}},
		{[]string{"key=a@b"}, 1, []string{"key=a@b"}, nil},
	}
	for _, tt := range tests {
		var sel selectionOptions
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		words, _, ok := parseRemote(fs, &sel, tt.args, tt.keep)
		if !ok || !slices.Equal(words, tt.wantWords) || !slices.Equal([]string(sel.hosts), tt.wantHosts) {
			t.Errorf("%v: words = %q, hosts = %q", tt.args, words, sel.hosts)
		}
	}
}

func TestDestinationOnPackagesList(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, handlers: map[int]sshtest.Handler{1: packageHandler(rpmPackages)}})
	dest := "tester@" + fleet.servers[0].Addr
	for _, args := range [][]string{
		{"packages", "list", "openssl", dest},
		{"packages", "list", dest, "openssl"},
		{"packages", "list", "openssl", "--host", dest}, // flags after a word
	} {
		stdout, stderr, code := runCLI(t, append(args, fleet.credentials()...)...)
		if code != exitcode.OK || !contains(stdout, "openssl") || contains(stdout, "nginx") {
			t.Errorf("%v: code = %v, stdout = %q, stderr = %s", args, code, stdout, stderr)
		}
	}
}

func TestDestinationOnOtherCommands(t *testing.T) {
	fleet := newFleet(t, fleetOptions{servers: 1, handlers: map[int]sshtest.Handler{1: packageHandler(rpmPackages)}})
	dest := "tester@" + fleet.servers[0].Addr
	for _, args := range [][]string{
		{"exec", dest, "--command", "echo ok"},
		{"uptime", dest},
		{"packages", "check", "openssl", dest},
	} {
		stdout, stderr, code := runCLI(t, append(args, fleet.credentials()...)...)
		if code != exitcode.OK {
			t.Errorf("%v: code = %v, stdout = %q, stderr = %s", args, code, stdout, stderr)
		}
	}
}

func TestSeveralServersPutTheNameFirst(t *testing.T) {
	inv := filepath.Join(t.TempDir(), "inv.yaml")
	fleet := newTestFleet(t, 2, nil)
	var b strings.Builder
	b.WriteString("servers:\n")
	for i, name := range []string{"web01", "server01"} {
		host, port, _ := net.SplitHostPort(fleet.servers[i].Addr)
		fmt.Fprintf(&b, "  - name: %s\n    address: %s\n    port: %s\n    user: tester\n", name, host, port)
	}
	if err := os.WriteFile(inv, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	args := append([]string{"exec", "--all", "--ordered", "--inventory", inv, "--command", "echo /boot"}, fleet.credentials()...)
	stdout, stderr, code := runCLI(t, args...)
	if code != exitcode.OK || !strings.HasPrefix(stdout, "web01___>: /boot\nserver01>: /boot\n") {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
	stdout, _, _ = runCLI(t, append(args, "--blocks")...)
	if !contains(stdout, "=== web01: ok") {
		t.Errorf("--blocks: stdout = %q", stdout)
	}
}

func TestOneServerKeepsItsBlock(t *testing.T) {
	fleet := newTestFleet(t, 1, nil)
	args := append([]string{"exec", "--host", "tester@" + fleet.servers[0].Addr, "--command", "echo x"}, fleet.credentials()...)
	if stdout, _, _ := runCLI(t, args...); !contains(stdout, ": ok (exit 0") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestCommandAfterHost(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	hosts := "tester@" + fleet.servers[0].Addr + ",tester@" + fleet.servers[1].Addr
	args := append(fleet.credentials(), "--host", hosts, "ls", "-l", "/tmp")
	_, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code == exitcode.Usage {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	for _, s := range fleet.servers {
		if cmds := s.Commands(); len(cmds) != 1 || cmds[0] != "ls -l /tmp" {
			t.Errorf("remote command = %q", cmds)
		}
	}
}

func TestKnownNameAfterHostIsAServer(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	args := append(fleet.args(), "--host", "s1", "s2", "echo", "ok")
	stdout, stderr, code := runCLICtx(t, context.Background(), nil, args...)
	if code != exitcode.OK || strings.Count(stdout, ": ok\n") != 2 {
		t.Errorf("code = %v, stdout = %q, stderr = %s", code, stdout, stderr)
	}
}
