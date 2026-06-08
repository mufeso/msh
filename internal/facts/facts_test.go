package facts

import (
	"os/exec"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	f := Parse("Welcome to the server!\nos_id=rocky\nos_like=rhel centos fedora\nos_name=Rocky Linux 9.4\ncpus=8\nbad line\n")
	if f["os_family"] != "redhat" || f["cpus"] != "8" || f["os_name"] != "Rocky Linux 9.4" {
		t.Errorf("facts = %v", f)
	}
	if _, ok := f["Welcome to the server!"]; ok {
		t.Error("noise became a fact")
	}
}

func TestFamily(t *testing.T) {
	tests := []struct{ id, like, kernel, want string }{
		{"ubuntu", "debian", "Linux", "debian"},
		{"pop", "ubuntu debian", "Linux", "debian"},
		{"almalinux", "", "Linux", "redhat"},
		{"opensuse-leap", "suse", "Linux", "suse"},
		{"", "", "Darwin", "darwin"},
		{"mystery", "", "Linux", "unknown"},
	}
	for _, tt := range tests {
		if got := family(tt.id, tt.like, tt.kernel); got != tt.want {
			t.Errorf("family(%q, %q) = %q, want %q", tt.id, tt.like, got, tt.want)
		}
	}
}

func TestScriptRunsInARealShell(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-s", "--")
	cmd.Stdin = strings.NewReader(Script)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	f := Parse(string(out))
	for _, key := range []string{"kernel", "arch", "package_manager", "init"} {
		if f[key] == "" {
			t.Errorf("fact %q is empty: %v", key, f)
		}
	}
}
