package history

import (
	"reflect"
	"testing"

	"example.com/msh/internal/output"
)

func TestDiff(t *testing.T) {
	a := Record{Results: []output.ResultView{
		{Server: "same", Status: "ok", Stdout: "x\n", DurationMS: 5},
		{Server: "changed", Status: "ok", Stdout: "1\n2\n3\n"},
		{Server: "gone", Status: "ok"},
	}}
	b := Record{Results: []output.ResultView{
		{Server: "same", Status: "ok", Stdout: "x\n", DurationMS: 900},
		{Server: "changed", Status: "failed", ExitCode: 2, Stdout: "1\nTWO\n3\n"},
		{Server: "new", Status: "ok"},
	}}
	want := []Change{
		{"changed", "status", "ok", "failed"},
		{"changed", "exit_code", "0", "2"},
		{"changed", "stdout", "2", "TWO"},
		{"gone", "presence", "in run", "not in run"},
		{"new", "presence", "not in run", "in run"},
	}
	if got := Diff(a, b); !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %+v", got)
	}
	if got := Diff(a, a); got != nil {
		t.Errorf("Diff with itself = %+v", got)
	}
}
