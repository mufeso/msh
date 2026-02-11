package task

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Set is a validated collection of tasks with an alias index.
type Set struct {
	tasks   map[string]*Task
	aliases map[string]string // alias -> task name
}

// NewSet validates the tasks of a "tasks:" map. Map keys become task
// names. All problems are reported together.
func NewSet(m map[string]Task) (*Set, error) {
	s := &Set{tasks: map[string]*Task{}, aliases: map[string]string{}}
	var problems []string

	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic error order

	for _, name := range names {
		t := m[name] // a copy of the map value
		t.Name = name
		if !namePattern.MatchString(name) {
			problems = append(problems, fmt.Sprintf("task %q: name must match %s", name, namePattern))
			continue
		}
		if err := t.validate(); err != nil {
			problems = append(problems, fmt.Sprintf("task %q: %v", name, err))
			continue
		}
		s.tasks[name] = &t
	}
	for _, name := range names {
		t, ok := s.tasks[name]
		if !ok {
			continue
		}
		for _, alias := range t.Aliases {
			if other, dup := s.aliases[alias]; dup {
				problems = append(problems, fmt.Sprintf("task %q: alias %q is already used by %q", name, alias, other))
				continue
			}
			if _, clash := m[alias]; clash {
				problems = append(problems, fmt.Sprintf("task %q: alias %q is also a task name", name, alias))
				continue
			}
			s.aliases[alias] = name
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid tasks:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return s, nil
}

// validate checks one task.
func (t *Task) validate() error {
	if err := t.validateShape(); err != nil {
		return err
	}
	if t.Mode() != ModeDirect {
		if _, err := LookupInterpreter(t.Interpreter); err != nil {
			return err
		}
	}
	return t.validatePlaceholders()
}

// Get returns a task by name or alias.
func (s *Set) Get(nameOrAlias string) (*Task, bool) {
	if s == nil {
		return nil, false
	}
	if t, ok := s.tasks[nameOrAlias]; ok {
		return t, true
	}
	if name, ok := s.aliases[nameOrAlias]; ok {
		return s.tasks[name], true
	}
	return nil, false
}

// List returns all tasks sorted by name.
func (s *Set) List() []*Task {
	if s == nil {
		return nil
	}
	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns task names and aliases, for suggestions.
func (s *Set) Names() []string {
	var out []string
	for _, t := range s.List() {
		out = append(out, t.Name)
		out = append(out, t.Aliases...)
	}
	return out
}
