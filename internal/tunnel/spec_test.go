package tunnel

import "testing"

func TestParseForward(t *testing.T) {
	tests := map[string]Spec{
		"8080:web01:80":                 {"127.0.0.1", 8080, "web01", 80},
		"0:db.example.com:5432":         {"127.0.0.1", 0, "db.example.com", 5432},
		"192.0.2.1:8080:localhost:80":   {"192.0.2.1", 8080, "localhost", 80},
		"*:8080:localhost:80":           {"0.0.0.0", 8080, "localhost", 80},
		":8080:localhost:80":            {"0.0.0.0", 8080, "localhost", 80},
		"8080:[2001:db8::1]:80":         {"127.0.0.1", 8080, "2001:db8::1", 80},
		"[::1]:8080:[2001:db8::1]:8443": {"::1", 8080, "2001:db8::1", 8443},
	}
	for in, want := range tests {
		if got, err := ParseForward(in); err != nil || got != want {
			t.Errorf("ParseForward(%q) = %+v, %v", in, got, err)
		}
	}
	for _, bad := range []string{
		"", "8080", "8080:web01", "a:web01:80", "8080:web01:x", "8080:web01:0",
		"70000:web01:80", "8080:web01:70000", "8080:-web:80", "8080:a b:80",
		"8080:[::1:80", "/tmp/s:web01:80", "1:2:3:4:5",
	} {
		if got, err := ParseForward(bad); err == nil {
			t.Errorf("ParseForward(%q) = %+v, want an error", bad, got)
		}
	}
}

func TestParseDynamic(t *testing.T) {
	for in, want := range map[string]Spec{
		"1080":        {BindHost: "127.0.0.1", BindPort: 1080},
		"192.0.2.1:9": {BindHost: "192.0.2.1", BindPort: 9},
		"*:1080":      {BindHost: "0.0.0.0", BindPort: 1080},
		"[::1]:1080":  {BindHost: "::1", BindPort: 1080},
	} {
		if got, err := ParseDynamic(in); err != nil || got != want {
			t.Errorf("ParseDynamic(%q) = %+v, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "x", "1:2:3", "70000"} {
		if _, err := ParseDynamic(bad); err == nil {
			t.Errorf("ParseDynamic(%q) was accepted", bad)
		}
	}
}

func TestParseTarget(t *testing.T) {
	if got, err := ParseTarget("[2001:db8::1]:22"); err != nil || got != "[2001:db8::1]:22" {
		t.Errorf("got %q, %v", got, err)
	}
	for _, bad := range []string{"web01", "web01:0", "web01:x", "-x:22"} {
		if _, err := ParseTarget(bad); err == nil {
			t.Errorf("ParseTarget(%q) was accepted", bad)
		}
	}
}

func TestPublic(t *testing.T) {
	for host, want := range map[string]bool{
		"127.0.0.1": false, "::1": false, "localhost": false,
		"0.0.0.0": true, "192.0.2.1": true, "example.com": true,
	} {
		if got := (Spec{BindHost: host}).Public(); got != want {
			t.Errorf("%s: Public = %v", host, got)
		}
	}
}

func TestSpecAddresses(t *testing.T) {
	s := Spec{"::1", 8080, "2001:db8::1", 80}
	if s.BindAddr() != "[::1]:8080" || s.Target() != "[2001:db8::1]:80" {
		t.Errorf("%s, %s", s.BindAddr(), s.Target())
	}
}
