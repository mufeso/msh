package tmux

import (
	"errors"
	"testing"

	"example.com/msh/internal/errs"
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
