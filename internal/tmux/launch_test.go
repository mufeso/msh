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
