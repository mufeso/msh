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
