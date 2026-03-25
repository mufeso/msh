package tmux

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"

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

// sessionNameRE limits session names to characters that are safe as a
// tmux target (":" and "." have special meanings there).
var sessionNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidSessionName reports whether name can be used as a session name.
func ValidSessionName(name string) bool {
	return sessionNameRE.MatchString(name)
}

// RandomToken returns 8 random hex characters.
func RandomToken() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%08x", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// HasSession reports whether a session with exactly this name exists.
func HasSession(ctx context.Context, r Runner, name string) bool {
	// "=" makes tmux match the name exactly instead of by prefix.
	_, err := r.Output(ctx, "has-session", "-t", "="+name)
	return err == nil
}

// Layouts are the tmux layouts msh accepts.
var Layouts = []string{"tiled", "even-horizontal", "even-vertical", "main-horizontal", "main-vertical"}

// ValidLayout reports whether name is one of Layouts.
func ValidLayout(name string) bool {
	return slices.Contains(Layouts, name)
}

// Plan describes a tmux session to create.
type Plan struct {
	Session string // session name
	Owner   string // value for the @msh-owner option
	Layout  string // final layout; "" means tiled
	Panes   []Pane
}

// Launch creates a new detached session for plan and returns the pane
// IDs in plan order. It never modifies an existing session.
func Launch(ctx context.Context, r Runner, plan Plan) ([]string, error) {
	if len(plan.Panes) == 0 {
		return nil, errors.New("no panes to open")
	}
	if !ValidSessionName(plan.Session) {
		return nil, errs.New(errs.KindUsage, "", fmt.Errorf("invalid tmux session name %q (use letters, digits, - and _)", plan.Session))
	}
	if HasSession(ctx, r, plan.Session) {
		return nil, errs.New(errs.KindTmuxStartup, "", fmt.Errorf("a tmux session named %q already exists; msh never reuses or modifies existing sessions", plan.Session))
	}
	first := plan.Panes[0]
	id, err := r.Output(ctx, "new-session", "-d", "-s", plan.Session, "-n", "msh", "-P", "-F", "#{pane_id}", first.Command)
	if err != nil {
		return nil, errs.New(errs.KindTmuxStartup, "", fmt.Errorf("creating session %q: %w", plan.Session, err))
	}
	if _, err := r.Output(ctx, "set-option", "-t", "="+plan.Session, OwnerOption, plan.Owner); err != nil {
		return nil, errs.New(errs.KindTmuxStartup, "", err)
	}
	ids := []string{id}
	if err := title(ctx, r, id, first.Title); err != nil {
		return ids, err
	}
	// Show pane titles in the borders, so every pane names its server.
	if _, err := r.Output(ctx, "set-option", "-t", "="+plan.Session, "pane-border-status", "top"); err != nil {
		return ids, errs.New(errs.KindTmuxStartup, "", err)
	}
	for _, p := range plan.Panes[1:] {
		// CTRL+C stops launching new panes.
		if err := ctx.Err(); err != nil {
			return ids, errs.New(errs.KindCanceled, "", err)
		}
		paneID, err := r.Output(ctx, "split-window", "-d", "-t", ids[0], "-P", "-F", "#{pane_id}", p.Command)
		if err != nil {
			return ids, errs.New(errs.KindTmuxStartup, "", fmt.Errorf("opening pane for %s: %w", p.Title, err))
		}
		ids = append(ids, paneID)
		if err := title(ctx, r, paneID, p.Title); err != nil {
			return ids, err
		}
		// Re-tile after every split; otherwise the window runs out of
		// space for further splits.
		if _, err := r.Output(ctx, "select-layout", "-t", ids[0], "tiled"); err != nil {
			return ids, errs.New(errs.KindTmuxStartup, "", err)
		}
	}
	layout := plan.Layout
	if layout == "" {
		layout = "tiled"
	}
	if _, err := r.Output(ctx, "select-layout", "-t", ids[0], layout); err != nil {
		return ids, errs.New(errs.KindTmuxStartup, "", err)
	}
	return ids, nil
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

// title sets a pane's title.
func title(ctx context.Context, r Runner, paneID, t string) error {
	if _, err := r.Output(ctx, "select-pane", "-t", paneID, "-T", t); err != nil {
		return errs.New(errs.KindTmuxStartup, "", err)
	}
	return nil
}
