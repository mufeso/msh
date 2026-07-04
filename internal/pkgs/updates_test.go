package pkgs

import "testing"

func TestParseUpdatesDnf(t *testing.T) {
	out := "#tool=dnf\n\nnginx.x86_64    1:1.26.0-1.el9    appstream\nkernel.x86_64   5.14.0-500.el9    baseos\nObsoleting Packages\n"
	tool, u := ParseUpdates(out)
	if tool != "dnf" || len(u) != 2 || u[0].Name != "nginx" || u[0].Version != "1:1.26.0-1.el9" || u[1].Source != "baseos" {
		t.Errorf("tool = %s, updates = %+v", tool, u)
	}
}

func TestParseUpdatesApt(t *testing.T) {
	out := "#tool=apt\nListing...\nnginx/noble-updates 1.24.0-2ubuntu7.1 amd64 [upgradable from: 1.24.0-2ubuntu7]\n"
	tool, u := ParseUpdates(out)
	if tool != "apt" || len(u) != 1 || u[0].Name != "nginx" || u[0].Source != "noble-updates" {
		t.Errorf("tool = %s, updates = %+v", tool, u)
	}
}

func TestParseUpdatesNone(t *testing.T) {
	if tool, u := ParseUpdates("#tool=none\n"); tool != "none" || u != nil {
		t.Errorf("tool = %s, updates = %+v", tool, u)
	}
}
