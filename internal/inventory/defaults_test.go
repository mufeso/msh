package inventory

import "testing"

func TestApplyDefaultsPort(t *testing.T) {
	inv := &Inventory{Servers: []Server{
		{Name: "a", Address: "192.0.2.1"},
		{Name: "b", Address: "192.0.2.2", Port: 2222},
	}}
	inv.ApplyDefaults("local")
	if inv.Servers[0].Port != DefaultPort {
		t.Errorf("a.Port = %d, want %d", inv.Servers[0].Port, DefaultPort)
	}
	if inv.Servers[1].Port != 2222 {
		t.Errorf("b.Port = %d, want 2222 (explicit port must win)", inv.Servers[1].Port)
	}
}

func TestApplyDefaultsUserPrecedence(t *testing.T) {
	tests := []struct {
		name         string
		serverUser   string
		defaultsUser string
		want         string
	}{
		{"server wins", "srv", "def", "srv"},
		{"inventory default", "", "def", "def"},
		{"fallback", "", "", "local"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := &Inventory{
				Defaults: Defaults{User: tt.defaultsUser},
				Servers:  []Server{{Name: "a", Address: "192.0.2.1", User: tt.serverUser}},
			}
			inv.ApplyDefaults("local")
			if got := inv.Servers[0].User; got != tt.want {
				t.Errorf("User = %q, want %q", got, tt.want)
			}
		})
	}
}
