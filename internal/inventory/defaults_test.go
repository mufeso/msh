package inventory

import "testing"

func TestApplyDefaults(t *testing.T) {
	inv := &Inventory{Servers: []Server{
		{Name: "a", Address: "192.0.2.1"},
		{Name: "b", Address: "192.0.2.2", Port: 2222, User: "own"},
	}}
	inv.ApplyDefaults("resolved", 2200)

	if a := inv.Servers[0]; a.Port != 2200 || a.User != "resolved" {
		t.Errorf("a = %+v, want resolved defaults", a)
	}
	if b := inv.Servers[1]; b.Port != 2222 || b.User != "own" {
		t.Errorf("b = %+v, server values must win", b)
	}
}
