package pkgs

import "testing"

const rpmOutput = "Welcome!\n#manager=rpm\nnginx\t1:1.24.0-1.el9\tx86_64\nbash\t0:5.1.8-6.el9\tx86_64\ngpg-pubkey\t0:abc-def\t(none)\nglibc\t0:2.34-1\ti686\nglibc\t0:2.34-1\tx86_64\n"

func TestParseRPM(t *testing.T) {
	l := Parse(rpmOutput)
	if l.Manager != RPM || len(l.Packages) != 4 {
		t.Fatalf("list = %+v", l)
	}
	if got := l.Find("bash"); len(got) != 1 || got[0].Version != "5.1.8-6.el9" {
		t.Errorf("bash = %+v", got)
	}
	if got := l.Find("nginx"); got[0].Version != "1:1.24.0-1.el9" {
		t.Errorf("a real epoch must stay: %+v", got)
	}
	if len(l.Find("glibc")) != 2 || len(l.Find("gpg-pubkey")) != 0 {
		t.Errorf("packages = %+v", l.Packages)
	}
}

func TestParseNone(t *testing.T) {
	if l := Parse("#manager=none\n"); l.Manager != None || len(l.Packages) != 0 {
		t.Errorf("list = %+v", l)
	}
}
