package tmux

import (
	"context"
	"errors"
	"fmt"

	"example.com/msh/internal/errs"
)

// OwnerOption marks sessions created by msh; its value identifies the
// msh run. msh only ever kills sessions whose marker it set.
const OwnerOption = "@msh-owner"

// Pane is one pane to open.
type Pane struct {
	Title   string // shown in the pane border (the server name)
	Command string // shell command line, already quoted
}

// Plan describes a tmux session to create.
type Plan struct {
	Session string // session name
	Owner   string // value for the @msh-owner option
	Panes   []Pane
}

// Launch creates a new detached session for plan and returns the pane
// IDs in plan order. It never modifies an existing session.
func Launch(ctx context.Context, r Runner, plan Plan) ([]string, error) {
	if len(plan.Panes) == 0 {
		return nil, errors.New("no panes to open")
	}
	first := plan.Panes[0]
	id, err := r.Output(ctx, "new-session", "-d", "-s", plan.Session, "-n", "msh", "-P", "-F", "#{pane_id}", first.Command)
	if err != nil {
		return nil, errs.New(errs.KindTmuxStartup, "", fmt.Errorf("creating session %q: %w", plan.Session, err))
	}
	if _, err := r.Output(ctx, "set-option", "-t", "="+plan.Session, OwnerOption, plan.Owner); err != nil {
		return nil, errs.New(errs.KindTmuxStartup, "", err)
	}
	return []string{id}, nil
}

// Attach connects the user to the session: switch-client when msh
// already runs inside tmux, attach-session otherwise.
func Attach(ctx context.Context, r Runner, session string, insideTmux bool) error {
	if insideTmux {
		_, err := r.Output(ctx, "switch-client", "-t", "="+session)
		return err
	}
	return r.Attach(ctx, "attach-session", "-t", "="+session)
}
