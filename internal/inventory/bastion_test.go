package inventory

import (
	"strings"
	"testing"
)

func TestPickBastionPrecedence(t *testing.T) {
	tests := []struct {
		override, server, fallback, want string
	}{
		{"", "", "", ""},
		{"", "", "def", "def"},
		{"", "srv", "def", "srv"},
		{"cli", "srv", "def", "cli"},
		{"none", "srv", "def", ""},
		{"", "none", "def", ""},
		{"", "", "none", ""},
	}
	for _, tt := range tests {
		if got := pickBastion(tt.override, tt.server, tt.fallback); got != tt.want {
			t.Errorf("pickBastion(%q, %q, %q) = %q, want %q", tt.override, tt.server, tt.fallback, got, tt.want)
		}
	}
}

func bastionInventory() *Inventory {
	return &Inventory{
		Bastions: []Bastion{{Name: "b1", Address: "192.0.2.100"}, {Name: "b2", Address: "192.0.2.101"}},
		Servers: []Server{
			{Name: "a", Address: "10.0.0.1", Bastion: "b2"},
			{Name: "b", Address: "10.0.0.2"},
		},
	}
}

func TestResolveBastions(t *testing.T) {
	inv := bastionInventory()
	if err := inv.ResolveBastions("", "b1"); err != nil {
		t.Fatal(err)
	}
	if inv.Servers[0].Bastion != "b2" || inv.Servers[1].Bastion != "b1" {
		t.Errorf("servers = %+v", inv.Servers)
	}

	inv = bastionInventory()
	if err := inv.ResolveBastions("b1", ""); err != nil {
		t.Fatal(err)
	}
	if inv.Servers[0].Bastion != "b1" {
		t.Errorf("command line must win: %+v", inv.Servers[0])
	}

	if err := bastionInventory().ResolveBastions("nope", ""); err == nil {
		t.Error("unknown override accepted")
	}
}

func TestValidateBastions(t *testing.T) {
	inv := bastionInventory()
	inv.Bastions = append(inv.Bastions, Bastion{Name: "b1"}, Bastion{Name: "none", Address: "x"})
	inv.Servers = append(inv.Servers, Server{Name: "c", Address: "10.0.0.3", Bastion: "ghost"})
	err := inv.Validate()
	if err == nil {
		t.Fatal("Validate() = nil")
	}
	for _, want := range []string{`bastion "b1": duplicate`, `bastion "b1": address is required`, `"none" is reserved`, `unknown bastion "ghost"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
}
