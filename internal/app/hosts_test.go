package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
)

const fixture = "testdata/inventory.json"

// listJSON runs "hosts list" with extra args and decodes the JSON result.
func listJSON(t *testing.T, args ...string) []inventory.Server {
	t.Helper()
	all := append([]string{"hosts", "list", "--inventory", fixture, "--output", "json"}, args...)
	stdout, stderr, code := runCLI(t, all...)
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	var servers []inventory.Server
	if err := json.Unmarshal([]byte(stdout), &servers); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	return servers
}

func serverNames(servers []inventory.Server) []string {
	var out []string
	for _, s := range servers {
		out = append(out, s.Name)
	}
	return out
}

func TestHostsListSelections(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"everything by default", nil, []string{"web01", "web02", "db01"}},
		{"all", []string{"--all"}, []string{"web01", "web02", "db01"}},
		{"host list", []string{"--host", "db01,web02"}, []string{"db01", "web02"}},
		{"group", []string{"--group", "web"}, []string{"web01", "web02"}},
		{"two groups in given order", []string{"--group", "database", "--group", "staging"}, []string{"db01", "web02"}},
		{"tag", []string{"--tag", "environment=production"}, []string{"web01", "db01"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serverNames(listJSON(t, tt.args...))
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostsListAppliesDefaults(t *testing.T) {
	servers := listJSON(t, "--host", "web01")
	if servers[0].Port != 22 || servers[0].User != "admin" {
		t.Errorf("defaults not applied: %+v", servers[0])
	}
}

func TestHostsListGoldenTable(t *testing.T) {
	stdout, _, code := runCLI(t, "hosts", "list", "--inventory", fixture)
	if code != exitcode.OK {
		t.Fatalf("code = %v", code)
	}
	golden(t, "hosts_list", stdout)
}

func TestHostsListGeneratedInventory(t *testing.T) {
	dir := t.TempDir() // removed automatically when the test ends
	path := filepath.Join(dir, "inv.json")
	content := `{"servers":[{"name":"only","address":"192.0.2.99","user":"ops"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runCLI(t, "hosts", "list", "--inventory", path, "--output", "json")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	var servers []inventory.Server
	if err := json.Unmarshal([]byte(stdout), &servers); err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].Port != inventory.DefaultPort {
		t.Errorf("servers = %+v", servers)
	}
}

func TestHostsListYAMLInventory(t *testing.T) {
	stdout, stderr, code := runCLI(t, "hosts", "list", "--inventory", "testdata/inventory.yaml", "--group", "database")
	if code != exitcode.OK {
		t.Fatalf("code = %v, stderr = %s", code, stderr)
	}
	if !contains(stdout, "db01") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestHostFlagForms(t *testing.T) {
	file := filepath.Join(t.TempDir(), "servers.txt")
	if err := os.WriteFile(file, []byte("# production\ndb01\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"--host", "web01,web02,db01"},
		{"--host", "web01 web02 db01"},
		{"--host", "web01 web02", "--host", "@" + file},
		{"--host", "web01,web02", "--host", "@" + file},
	} {
		got := serverNames(listJSON(t, args...))
		if !slices.Equal(got, []string{"web01", "web02", "db01"}) {
			t.Errorf("%v: got %v", args, got)
		}
	}
	if _, _, code := runCLI(t, "hosts", "list", "--inventory", fixture, "--host", "@/does/not/exist"); code != exitcode.Usage {
		t.Errorf("missing file: code = %v", code)
	}
}

func TestHostsListPattern(t *testing.T) {
	if got := serverNames(listJSON(t, "web")); !slices.Equal(got, []string{"web01", "web02"}) {
		t.Errorf("web: %v", got)
	}
	if got := serverNames(listJSON(t, "^db")); !slices.Equal(got, []string{"db01"}) {
		t.Errorf("^db: %v", got)
	}
	if got := serverNames(listJSON(t, "WEB", "--ignore-case", "--group", "web")); len(got) != 2 {
		t.Errorf("WEB: %v", got)
	}
	if _, stderr, code := runCLI(t, "hosts", "list", "a", "b"); code != exitcode.Usage || !contains(stderr, "only one pattern") {
		t.Errorf("code = %v, stderr = %q", code, stderr)
	}
}
