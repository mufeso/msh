package facts

import (
	"context"
	"testing"
	"time"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/executor/fake"
	"example.com/msh/internal/inventory"
)

func TestCache(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	f := &fake.Executor{Default: fake.Response{Result: executor.Result{Status: executor.StatusOK, Stdout: "os_id=rocky\n"}}}
	g := &Gatherer{Exec: f, Workers: 2, CacheDir: t.TempDir(), TTL: time.Hour, Now: func() time.Time { return now }}
	s := []inventory.Server{{Name: "a", Address: "192.0.2.1"}}

	if found, _ := g.Gather(context.Background(), s); found["a"]["os_family"] != "redhat" {
		t.Fatalf("found = %v", found)
	}
	g.Gather(context.Background(), s)
	if len(f.Calls()) != 1 {
		t.Errorf("cached facts were not used: %d calls", len(f.Calls()))
	}

	now = now.Add(2 * time.Hour)
	g.Gather(context.Background(), s)
	if len(f.Calls()) != 2 {
		t.Errorf("expired facts were used: %d calls", len(f.Calls()))
	}

	g.Refresh = true
	g.Gather(context.Background(), s)
	if len(f.Calls()) != 3 {
		t.Errorf("--refresh was ignored: %d calls", len(f.Calls()))
	}
	g.Refresh = false

	s[0].Address = "192.0.2.2" // same name, different machine
	g.Gather(context.Background(), s)
	if len(f.Calls()) != 4 {
		t.Errorf("facts of another address were used: %d calls", len(f.Calls()))
	}
}
