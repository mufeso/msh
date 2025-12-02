package inventory

import "testing"

func TestApplyDefaultsPort(t *testing.T) {
	inv := &Inventory{Servers: []Server{
		{Name: "a", Address: "192.0.2.1"},
		{Name: "b", Address: "192.0.2.2", Port: 2222},
	}}
	inv.ApplyDefaults()
	if inv.Servers[0].Port != DefaultPort {
		t.Errorf("a.Port = %d, want %d", inv.Servers[0].Port, DefaultPort)
	}
	if inv.Servers[1].Port != 2222 {
		t.Errorf("b.Port = %d, want 2222 (explicit port must win)", inv.Servers[1].Port)
	}

	inv = &Inventory{Defaults: Defaults{Port: 2200}, Servers: []Server{{Name: "c", Address: "192.0.2.3"}}}
	inv.ApplyDefaults()
	if inv.Servers[0].Port != 2200 {
		t.Errorf("c.Port = %d, want inventory default 2200", inv.Servers[0].Port)
	}
}
