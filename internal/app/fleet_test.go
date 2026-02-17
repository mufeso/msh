package app

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/sshtest"
)

// testFleet is a set of in-process SSH servers plus an inventory, key and
// known_hosts file that describe them.
type testFleet struct {
	servers   []*sshtest.Server
	bastion   *sshtest.Server // nil unless fleetOptions.bastion
	key       sshtest.Key
	dir       string
	inventory string
	identity  string
	known     string
}

// fleetOptions configure newFleet.
type fleetOptions struct {
	servers  int
	handler  sshtest.Handler
	handlers map[int]sshtest.Handler // per-server handler by number (1-based)
	bastion  bool                    // route every server through a bastion named "jump"
}

// newTestFleet starts n servers named s1..sN. Servers with an odd number
// are in group "odd", the others in "even"; all are in "all".
func newTestFleet(t *testing.T, n int, handler sshtest.Handler) *testFleet {
	t.Helper()
	return newFleet(t, fleetOptions{servers: n, handler: handler})
}

func newFleet(t *testing.T, o fleetOptions) *testFleet {
	t.Helper()
	f := &testFleet{key: sshtest.NewKey(t), dir: t.TempDir()}
	authorized := []ssh.PublicKey{f.key.Public()}
	var knownLines []string
	var inv strings.Builder

	if o.bastion {
		f.bastion = sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: authorized, AllowForwarding: true})
		knownLines = append(knownLines, f.bastion.KnownHostsLine())
		host, port, _ := net.SplitHostPort(f.bastion.Addr)
		fmt.Fprintf(&inv, "bastions:\n  - name: jump\n    address: %s\n    port: %s\n    user: tester\n", host, port)
	}
	inv.WriteString("servers:\n")
	for i := 1; i <= o.servers; i++ {
		handler := o.handler
		if h, ok := o.handlers[i]; ok {
			handler = h
		}
		srv := sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: authorized, Handler: handler})
		f.servers = append(f.servers, srv)
		knownLines = append(knownLines, srv.KnownHostsLine())
		host, port, _ := net.SplitHostPort(srv.Addr)
		group := "even"
		if i%2 == 1 {
			group = "odd"
		}
		fmt.Fprintf(&inv, "  - name: s%d\n    address: %s\n    port: %s\n    user: tester\n    groups: [all, %s]\n", i, host, port, group)
		if o.bastion {
			inv.WriteString("    bastion: jump\n")
		}
	}
	f.inventory = filepath.Join(f.dir, "inventory.yaml")
	if err := os.WriteFile(f.inventory, []byte(inv.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	f.identity = sshtest.WriteKeyFile(t, f.dir, "id_ed25519", f.key, "")
	f.known = sshtest.WriteKnownHosts(t, f.dir, knownLines...)
	return f
}

// args returns the flags that point msh at this fleet.
func (f *testFleet) args() []string {
	return []string{"--inventory", f.inventory, "--identity", f.identity, "--known-hosts", f.known}
}

// run runs an msh subcommand against the fleet.
func (f *testFleet) run(t *testing.T, cmd []string, extra ...string) (string, string, int) {
	t.Helper()
	args := append(append(cmd, f.args()...), extra...)
	stdout, stderr, code := runCLI(t, args...)
	return stdout, stderr, int(code)
}
