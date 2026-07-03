package pkgs

import "testing"

func TestCheckPresence(t *testing.T) {
	l := Parse(rpmOutput)
	s, err := ParseSpec("nginx")
	if err != nil {
		t.Fatal(err)
	}
	if r := s.Check(l); r.Status != StatusOK || r.Version != "1:1.24.0-1.el9" {
		t.Errorf("result = %+v", r)
	}
	s, _ = ParseSpec("httpd")
	if r := s.Check(l); r.Status != StatusMissing {
		t.Errorf("result = %+v", r)
	}
	if _, err := ParseSpec("a b; rm"); err == nil {
		t.Error("invalid name accepted")
	}
}

func TestParseSpec(t *testing.T) {
	tests := map[string]Spec{
		"openssl>=3.0": {Name: "openssl", Op: ">=", Version: "3.0"},
		"a<=1":         {Name: "a", Op: "<=", Version: "1"},
		"a = 1.2-3":    {Name: "a", Op: "=", Version: "1.2-3"},
		"a!=2":         {Name: "a", Op: "!=", Version: "2"},
		"!telnet":      {Name: "telnet", Absent: true},
		"libstdc++":    {Name: "libstdc++"},
	}
	for in, want := range tests {
		got, err := ParseSpec(in)
		if err != nil || got != want {
			t.Errorf("ParseSpec(%q) = %+v, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "a>=", ">=1", "!", "a b"} {
		if _, err := ParseSpec(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestCheckVersions(t *testing.T) {
	rpm := Parse("#manager=rpm\nopenssl\t1:3.0.7-16.el9\tx86_64\nglibc\t0:2.34-1\ti686\nglibc\t0:2.34-9\tx86_64\n")
	deb := Parse("#manager=dpkg\nopenssl\t3.0.13-0ubuntu3\tamd64\n")
	tests := []struct {
		list *List
		spec string
		want string
	}{
		{rpm, "openssl>=3.0", StatusOK},
		{rpm, "openssl>=3.1", StatusWrongVersion},
		{deb, "openssl>=3.0.10", StatusOK},
		{deb, "openssl<3.0.2", StatusWrongVersion},
		{rpm, "glibc>=2.34-5", StatusOK},
		{rpm, "!telnet", StatusOK},
		{rpm, "!openssl", StatusUnwanted},
		{rpm, "nginx>=1", StatusMissing},
	}
	for _, tt := range tests {
		s, err := ParseSpec(tt.spec)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.Check(tt.list); got.Status != tt.want {
			t.Errorf("%s on %s: %+v, want %s", tt.spec, tt.list.Manager, got, tt.want)
		}
	}
}
