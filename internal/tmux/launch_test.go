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

func TestLaunchRefusesExistingSession(t *testing.T) {
	f := tmuxtest.NewFake("work")
	_, err := Launch(context.Background(), f, Plan{Session: "work", Owner: "o", Panes: []Pane{{Command: "x"}}})
	if err == nil || len(f.CallsTo("new-session")) != 0 {
		t.Errorf("err = %v, calls = %v", err, f.Calls())
	}
}

func TestSessionNames(t *testing.T) {
	for _, ok := range []string{"fleet", "msh-1a2b", "web_01"} {
		if !ValidSessionName(ok) {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"", "a:b", "a.b", "a b", "x;y", "$(id)"} {
		if ValidSessionName(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
	if a, b := RandomToken(), RandomToken(); a == b || len(a) != 8 {
		t.Errorf("tokens %q %q", a, b)
	}
}

func TestFailedLaunchRemovesOnlyItsSession(t *testing.T) {
	f := tmuxtest.NewFake("someone-else")
	f.FailOn = "split-window"
	_, err := Launch(context.Background(), f, Plan{Session: "mine", Owner: "run-7", Panes: []Pane{{Command: "a"}, {Command: "b"}}})
	if err == nil {
		t.Fatal("Launch succeeded despite a failing split")
	}
	if f.HasSession("mine") {
		t.Error("the half-built session was not removed")
	}
	if !f.HasSession("someone-else") {
		t.Error("an unrelated session was removed")
	}
}

func TestKillOwnedChecksMarker(t *testing.T) {
	f := tmuxtest.NewFake()
	f.Output(context.Background(), "new-session", "-d", "-s", "x")
	f.Output(context.Background(), "set-option", "-t", "=x", OwnerOption, "other-run")
	if err := KillOwned(context.Background(), f, "x", "my-run"); err == nil || !f.HasSession("x") {
		t.Errorf("killed a session owned by another run: err = %v", err)
	}
	if err := KillOwned(context.Background(), f, "x", "other-run"); err != nil || f.HasSession("x") {
		t.Errorf("owner could not kill: err = %v", err)
	}
}

func TestCancelDuringLaunchCleansUp(t *testing.T) {
	f := tmuxtest.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel as soon as the session exists (simulating CTRL+C mid-way).
	f.OnCall = func(args []string) {
		if args[0] == "select-pane" {
			cancel()
		}
	}
	_, err := Launch(ctx, f, Plan{Session: "s", Owner: "o", Panes: []Pane{{Command: "a"}, {Command: "b"}, {Command: "c"}}})
	if err == nil || f.HasSession("s") {
		t.Errorf("err = %v, sessions = %v", err, f.Sessions())
	}
}
