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
