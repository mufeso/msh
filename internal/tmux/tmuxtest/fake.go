// Package tmuxtest provides an in-memory tmux for tests.
package tmuxtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Fake is a tmux.Runner that simulates sessions, panes and options.
type Fake struct {
	// FailOn makes the named tmux command (e.g. "split-window") fail.
	FailOn string

	mu       sync.Mutex
	calls    [][]string
	sessions map[string]map[string]string // session -> options
	panes    map[string]string            // pane id -> session
	nextPane int
	attached []string
}

// NewFake returns an empty fake with optional pre-existing sessions
// (which belong to "someone else").
func NewFake(existing ...string) *Fake {
	f := &Fake{sessions: map[string]map[string]string{}, panes: map[string]string{}}
	for _, s := range existing {
		f.sessions[s] = map[string]string{}
	}
	return f
}

// Output implements tmux.Runner.
func (f *Fake) Output(_ context.Context, args ...string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string(nil), args...))
	if len(args) == 0 {
		return "", errors.New("no command")
	}
	if args[0] == f.FailOn {
		return "", fmt.Errorf("tmux %s: simulated failure", args[0])
	}
	switch args[0] {
	case "-V":
		return "tmux 3.4", nil
	case "has-session":
		if _, ok := f.sessions[target(args)]; ok {
			return "", nil
		}
		return "", errors.New("can't find session")
	case "new-session":
		name := flagValue(args, "-s")
		if _, ok := f.sessions[name]; ok {
			return "", errors.New("duplicate session")
		}
		f.sessions[name] = map[string]string{}
		return f.newPane(name), nil
	case "split-window":
		session := f.panes[flagValue(args, "-t")]
		if session == "" {
			return "", errors.New("can't find pane")
		}
		return f.newPane(session), nil
	case "set-option":
		opts, ok := f.sessions[target(args)]
		if !ok {
			return "", errors.New("can't find session")
		}
		opts[args[len(args)-2]] = args[len(args)-1]
		return "", nil
	case "show-options":
		opts, ok := f.sessions[target(args)]
		if !ok {
			return "", errors.New("can't find session")
		}
		return opts[args[len(args)-1]], nil
	case "kill-session":
		delete(f.sessions, target(args))
		return "", nil
	default: // select-layout, select-pane, switch-client, ...
		return "", nil
	}
}

// Attach implements tmux.Runner.
func (f *Fake) Attach(_ context.Context, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string(nil), args...))
	f.attached = append(f.attached, target(args))
	return nil
}

func (f *Fake) newPane(session string) string {
	f.nextPane++
	id := fmt.Sprintf("%%%d", f.nextPane)
	f.panes[id] = session
	return id
}

// Calls returns every recorded call.
func (f *Fake) Calls() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.calls...)
}

// CallsTo returns the calls of one tmux command.
func (f *Fake) CallsTo(command string) [][]string {
	var out [][]string
	for _, c := range f.Calls() {
		if len(c) > 0 && c[0] == command {
			out = append(out, c)
		}
	}
	return out
}

// Sessions returns the names of existing sessions.
func (f *Fake) Sessions() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for s := range f.sessions {
		out = append(out, s)
	}
	return out
}

// HasSession reports whether a session exists.
func (f *Fake) HasSession(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.sessions[name]
	return ok
}

// Attached returns the sessions that were attached.
func (f *Fake) Attached() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.attached...)
}

// target returns the value of -t without tmux's "=" exact-match prefix.
func target(args []string) string {
	return strings.TrimPrefix(flagValue(args, "-t"), "=")
}

func flagValue(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}
