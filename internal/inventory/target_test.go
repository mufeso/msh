package inventory

import "testing"

func TestParseTarget(t *testing.T) {
	tests := map[string]Target{
		"server1":                  {Host: "server1"},
		"ops@server1":              {User: "ops", Host: "server1"},
		"root@db01.example.com:22": {User: "root", Host: "db01.example.com", Port: 22},
		"192.0.2.10:2222":          {Host: "192.0.2.10", Port: 2222},
		"2001:db8::1":              {Host: "2001:db8::1"},
		"[2001:db8::1]":            {Host: "2001:db8::1"},
		"admin@[2001:db8::1]:2200": {User: "admin", Host: "2001:db8::1", Port: 2200},
		"ssh://ops@web01:2222/":    {User: "ops", Host: "web01", Port: 2222},
	}
	for in, want := range tests {
		got, err := ParseTarget(in)
		if err != nil || got != want {
			t.Errorf("ParseTarget(%q) = %+v, %v", in, got, err)
		}
	}
	for _, bad := range []string{
		"", "@host", "user@", "-oProxyCommand=x", "user@-host", "-user@host",
		"host:0", "host:99999", "host:ssh", "[::1", "[::1]x", "a b", "a;b", "$(id)",
	} {
		if got, err := ParseTarget(bad); err == nil {
			t.Errorf("ParseTarget(%q) = %+v, want an error", bad, got)
		}
	}
}

func TestAddTargets(t *testing.T) {
	inv := sampleInventory()
	d := TargetDefaults{User: "me", Port: 22}
	names, err := inv.AddTargets([]string{"web01", "root@db01", "ops@web02:2200", "x@198.51.100.7:2222", "new.example.com"}, d, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := inv.Select(Selector{Hosts: names})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("selected = %+v", got)
	}
	if db := got[1]; db.Name != "db01" || db.User != "root" || db.Address != "db01.example.com" {
		t.Errorf("db01 = %+v", db)
	}
	if w := got[2]; w.Port != 2200 || w.User != "ops" {
		t.Errorf("web02 = %+v", w)
	}
	if x := got[3]; x.Name != "x@198.51.100.7:2222" || x.Address != "198.51.100.7" || x.User != "x" || x.Port != 2222 {
		t.Errorf("ad hoc = %+v", x)
	}
	if n := got[4]; n.User != "me" || n.Port != 22 {
		t.Errorf("defaults = %+v", n)
	}
	if _, err := inv.AddTargets([]string{"-oProxyCommand=x"}, d, nil); err == nil {
		t.Error("an option-like target was accepted")
	}
}

func TestAddTargetsMatchesAddress(t *testing.T) {
	inv := sampleInventory()
	names, err := inv.AddTargets([]string{"admin@192.0.2.10"}, TargetDefaults{}, nil)
	if err != nil || names[0] != "web01" {
		t.Errorf("names = %v, err = %v", names, err)
	}
}

func TestAddTargetsKeepsPortsApart(t *testing.T) {
	inv := &Inventory{}
	names, err := inv.AddTargets([]string{"a@192.0.2.1:2201", "a@192.0.2.1:2202"}, TargetDefaults{}, nil)
	if err != nil || len(names) != 2 || len(inv.Servers) != 2 {
		t.Errorf("names = %v, servers = %+v, err = %v", names, inv.Servers, err)
	}
}
