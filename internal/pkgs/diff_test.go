package pkgs

import (
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {
	a := Parse("#manager=rpm\nnginx\t0:1.24-1\tx86_64\nbash\t0:5.1-1\tx86_64\nonly-a\t0:1-1\tx86_64\n")
	b := Parse("#manager=rpm\nnginx\t0:1.26-1\tx86_64\nbash\t0:5.1-1\tx86_64\nonly-b\t0:2-1\tx86_64\n")
	want := []Difference{
		{Package: "nginx", Baseline: "1.24-1", Version: "1.26-1"},
		{Package: "only-a", Baseline: "1-1"},
		{Package: "only-b", Version: "2-1"},
	}
	if got := Diff(a, b); !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %+v", got)
	}
	if got := Diff(a, a); got != nil {
		t.Errorf("Diff with itself = %+v", got)
	}
}

func TestDiffMultiArch(t *testing.T) {
	a := Parse("#manager=rpm\nglibc\t0:2.34-1\ti686\nglibc\t0:2.34-1\tx86_64\n")
	b := Parse("#manager=rpm\nglibc\t0:2.34-1\tx86_64\n")
	if got := Diff(a, b); got != nil {
		t.Errorf("one version on two architectures is no difference: %+v", got)
	}
}
