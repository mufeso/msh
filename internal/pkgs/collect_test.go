package pkgs

import (
	"context"
	"testing"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/executor/fake"
	"example.com/msh/internal/inventory"
)

func TestCollect(t *testing.T) {
	f := &fake.Executor{Responses: map[string]fake.Response{
		"a": {Result: executor.Result{Stdout: rpmOutput}},
		"b": {Result: executor.Result{Status: executor.StatusError}},
		"c": {Result: executor.Result{Stdout: "#manager=rpm\n", StdoutTruncated: true}},
	}}
	servers := []inventory.Server{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	found, failed := Collect(context.Background(), f, servers, 3, 0)
	if len(found) != 1 || found["a"].Manager != RPM || len(failed) != 2 {
		t.Errorf("found = %v, failed = %+v", found, failed)
	}
	if calls := f.Calls(); calls[0].Request.OutputLimit != outputLimit {
		t.Errorf("limit = %d", calls[0].Request.OutputLimit)
	}
}
