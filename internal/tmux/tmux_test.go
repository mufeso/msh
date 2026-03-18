package tmux

import (
	"context"
	"errors"
	"testing"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/tmux/tmuxtest"
)

func TestFind(t *testing.T) {
	found := func(string) (string, error) { return "/usr/bin/tmux", nil }
	if p, err := Find(found); err != nil || p != "/usr/bin/tmux" {
		t.Errorf("Find = %q, %v", p, err)
	}
	missing := func(string) (string, error) { return "", errors.New("not found") }
	if _, err := Find(missing); errs.KindOf(err) != errs.KindTmuxUnavailable {
		t.Errorf("err = %v, want tmux unavailable", err)
	}
}

func TestVersionWithFake(t *testing.T) {
	f := tmuxtest.NewFake()
	v, err := Version(context.Background(), f)
	if err != nil || v != "tmux 3.4" {
		t.Errorf("Version = %q, %v", v, err)
	}
	if calls := f.Calls(); len(calls) != 1 || calls[0][0] != "-V" {
		t.Errorf("calls = %v", calls)
	}
}

func TestExecRunnerUsesArgumentSlices(t *testing.T) {
	// /bin/echo stands in for tmux: the argument with spaces and quotes
	// must arrive as one word, untouched by any shell.
	r := ExecRunner{Path: "/bin/echo"}
	out, err := r.Output(context.Background(), "new-session", "-s", "a b'c; rm -rf /")
	if err != nil {
		t.Fatal(err)
	}
	if out != "new-session -s a b'c; rm -rf /" {
		t.Errorf("out = %q", out)
	}
}
