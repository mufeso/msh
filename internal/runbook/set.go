package runbook

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"example.com/msh/internal/task"
)

// maxRetries bounds retries so a typo cannot loop for hours.
const maxRetries = 100

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Set is a validated collection of runbooks.
type Set struct {
	books map[string]*Runbook
}

// NewSet validates a "runbooks:" map against the configured tasks. All
// problems are reported together.
func NewSet(m map[string]Runbook, tasks *task.Set) (*Set, error) {
	s := &Set{books: map[string]*Runbook{}}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	var problems []string
	for _, name := range names {
		rb := m[name]
		rb.Name = name
		if !namePattern.MatchString(name) {
			problems = append(problems, fmt.Sprintf("runbook %q: name must match %s", name, namePattern))
			continue
		}
		if err := rb.validate(tasks); err != nil {
			problems = append(problems, fmt.Sprintf("runbook %q: %v", name, err))
			continue
		}
		s.books[name] = &rb
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid runbooks:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return s, nil
}

// validate checks one runbook.
func (rb *Runbook) validate(tasks *task.Set) error {
	if len(rb.Steps) == 0 {
		return errors.New("at least one step is required")
	}
	// A small helper task gives runbooks the same argument rules as tasks.
	inputs, err := task.NewSet(map[string]task.Task{rb.Name: {Command: "true", Arguments: rb.Arguments}})
	if err != nil {
		return err
	}
	rb.inputs, _ = inputs.Get(rb.Name)
	seen := map[string]bool{}
	for i, s := range rb.Steps {
		if s.Name == "" {
			return fmt.Errorf("step %d: name is required", i+1)
		}
		if seen[s.Name] {
			return fmt.Errorf("step %q: duplicate name", s.Name)
		}
		seen[s.Name] = true
		if err := rb.validateStep(s, tasks); err != nil {
			return fmt.Errorf("step %q: %w", s.Name, err)
		}
	}
	return nil
}

func (rb *Runbook) validateStep(s Step, tasks *task.Set) error {
	if (s.Task == "") == (s.Command == "") {
		return errors.New("exactly one of task or command must be set")
	}
	if s.Retries < 0 || s.Retries > maxRetries {
		return fmt.Errorf("retries must be between 0 and %d", maxRetries)
	}
	if s.Command != "" {
		if len(s.Params) > 0 || len(s.Args) > 0 {
			return errors.New("params and args are only for task steps")
		}
		// A command step is an anonymous task: the task validator applies
		// the interpreter and shell-template rules.
		_, err := task.NewSet(map[string]task.Task{"step": rb.commandTask(s)})
		return err
	}
	t, ok := tasks.Get(s.Task)
	if !ok {
		return fmt.Errorf("unknown task %q", s.Task)
	}
	if t.Interactive {
		return fmt.Errorf("task %q is interactive; runbook steps run unattended", s.Task)
	}
	// Every {{name}} in params and args must be a runbook argument.
	for _, v := range s.Params {
		if _, err := rb.inputs.ExpandPlain(v, nil); err != nil {
			return err
		}
	}
	for _, v := range s.Args {
		if _, err := rb.inputs.ExpandPlain(v, nil); err != nil {
			return err
		}
	}
	return nil
}

// commandTask returns the anonymous task for a command step.
func (rb *Runbook) commandTask(s Step) task.Task {
	return task.Task{
		Name:             "step",
		Interpreter:      s.Interpreter,
		Command:          s.Command,
		AllowRedirection: s.AllowRedirection,
		Arguments:        rb.Arguments,
	}
}

// Get returns a runbook by name.
func (s *Set) Get(name string) (*Runbook, bool) {
	if s == nil {
		return nil, false
	}
	rb, ok := s.books[name]
	return rb, ok
}

// List returns all runbooks sorted by name.
func (s *Set) List() []*Runbook {
	if s == nil {
		return nil
	}
	out := make([]*Runbook, 0, len(s.books))
	for _, rb := range s.books {
		out = append(out, rb)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns all runbook names, for suggestions.
func (s *Set) Names() []string {
	var out []string
	for _, rb := range s.List() {
		out = append(out, rb.Name)
	}
	return out
}
