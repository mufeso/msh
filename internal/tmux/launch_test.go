package tmux

import (
	"context"
	"testing"

	"example.com/msh/internal/tmux/tmuxtest"
)

func TestLaunchCreatesOwnedSession(t *testing.T) {
	f := tmuxtest.NewFake()
	ids, err := Launch(context.Background(), f, Plan{Session: "msh-test", Owner: "run-1", Panes: []Pane{{Title: "s1", Command: "msh shell --host s1"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || !f.HasSession("msh-test") {
		t.Fatalf("ids = %v, sessions = %v", ids, f.Sessions())
	}
	owner, _ := f.Output(context.Background(), "show-options", "-v", "-t", "=msh-test", OwnerOption)
	if owner != "run-1" {
		t.Errorf("owner = %q", owner)
	}
}

func TestAttach(t *testing.T) {
	f := tmuxtest.NewFake("x")
	if err := Attach(context.Background(), f, "x", false); err != nil {
		t.Fatal(err)
	}
	if err := Attach(context.Background(), f, "x", true); err != nil {
		t.Fatal(err)
	}
	if len(f.Attached()) != 1 || len(f.CallsTo("switch-client")) != 1 {
		t.Errorf("calls = %v", f.Calls())
	}
}

func TestLaunchOnePanePerServer(t *testing.T) {
	f := tmuxtest.NewFake()
	panes := []Pane{{Title: "a", Command: "ca"}, {Title: "b", Command: "cb"}, {Title: "c", Command: "cc"}}
	ids, err := Launch(context.Background(), f, Plan{Session: "s", Owner: "o", Panes: panes})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] == ids[1] {
		t.Errorf("ids = %v", ids)
	}
	splits := f.CallsTo("split-window")
	if len(splits) != 2 || splits[0][len(splits[0])-1] != "cb" || splits[1][len(splits[1])-1] != "cc" {
		t.Errorf("splits = %v", splits)
	}
}

func TestLaunchStopsOnCancel(t *testing.T) {
	f := tmuxtest.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Launch(ctx, f, Plan{Session: "s", Owner: "o", Panes: []Pane{{Command: "a"}, {Command: "b"}}})
	if err == nil || len(f.CallsTo("split-window")) != 0 {
		t.Errorf("err = %v, calls = %v", err, f.Calls())
	}
}

func TestPaneTitles(t *testing.T) {
	f := tmuxtest.NewFake()
	ids, err := Launch(context.Background(), f, Plan{Session: "s", Owner: "o", Panes: []Pane{{Title: "web01", Command: "a"}, {Title: "web02", Command: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	titles := map[string]string{}
	for _, c := range f.CallsTo("select-pane") {
		titles[c[2]] = c[4]
	}
	if titles[ids[0]] != "web01" || titles[ids[1]] != "web02" {
		t.Errorf("titles = %v for panes %v", titles, ids)
	}
}
