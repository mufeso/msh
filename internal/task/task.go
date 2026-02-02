// Package task models reusable, named remote commands ("tasks").
//
// A task runs in exactly one of three modes:
//
//   - direct: Argv is a program and its arguments. No shell interprets
//     anything; argument values can never become shell syntax.
//   - shell:  Command is a shell command line (pipelines, globbing) run by
//     an explicitly configured shell. Argument values are inserted as
//     single-quoted shell words.
//   - script: Script is a (multi-line) program streamed to the
//     interpreter's standard input. Argument values are passed as
//     positional parameters ("$1", sys.argv[1]) and never inserted into
//     the script text.
package task

import (
	"errors"
	"fmt"

	"example.com/msh/internal/config"
)

// Mode is how a task is executed.
type Mode string

// Execution modes.
const (
	ModeDirect Mode = "direct"
	ModeShell  Mode = "shell"
	ModeScript Mode = "script"
)

// Safety classifies what a task may do to a server.
type Safety string

// Safety classes. Tasks that are not read-only require explicit
// confirmation (--yes) when run.
const (
	SafetyReadOnly  Safety = "read-only"
	SafetyChanges   Safety = "changes"
	SafetyDangerous Safety = "dangerous"
)

// Argument describes one task argument.
type Argument struct {
	Name        string  `json:"name" yaml:"name"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Default     *string `json:"default,omitempty" yaml:"default,omitempty"`
	Pattern     string  `json:"pattern,omitempty" yaml:"pattern,omitempty"` // optional regular expression the value must match
}

// Task is one named task.
type Task struct {
	Name        string           `json:"name" yaml:"-"` // the key in the "tasks:" map
	Description string           `json:"description,omitempty" yaml:"description,omitempty"`
	Aliases     []string         `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Interpreter string           `json:"interpreter,omitempty" yaml:"interpreter,omitempty"`
	Command     string           `json:"command,omitempty" yaml:"command,omitempty"`
	Argv        []string         `json:"argv,omitempty" yaml:"argv,omitempty"`
	Script      string           `json:"script,omitempty" yaml:"script,omitempty"`
	Arguments   []Argument       `json:"arguments,omitempty" yaml:"arguments,omitempty"`
	Interactive bool             `json:"interactive,omitempty" yaml:"interactive,omitempty"`
	Timeout     *config.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Output      string           `json:"output,omitempty" yaml:"output,omitempty"`
	Safety      Safety           `json:"safety,omitempty" yaml:"safety,omitempty"`

	// AllowRedirection permits <, > and >> in shell commands. Redirection
	// can overwrite files, so it must be enabled explicitly.
	AllowRedirection bool `json:"allow_redirection,omitempty" yaml:"allow_redirection,omitempty"`
}

// Mode returns the execution mode implied by the fields that are set.
func (t *Task) Mode() Mode {
	switch {
	case len(t.Argv) > 0:
		return ModeDirect
	case t.Script != "":
		return ModeScript
	default:
		return ModeShell
	}
}

// EffectiveSafety returns the safety class, defaulting to read-only.
func (t *Task) EffectiveSafety() Safety {
	if t.Safety == "" {
		return SafetyReadOnly
	}
	return t.Safety
}

// validateShape checks that exactly one of command, argv and script is
// set and that enumerated fields have known values.
func (t *Task) validateShape() error {
	set := 0
	for _, present := range []bool{t.Command != "", len(t.Argv) > 0, t.Script != ""} {
		if present {
			set++
		}
	}
	if set != 1 {
		return errors.New("exactly one of command, argv or script must be set")
	}
	switch t.EffectiveSafety() {
	case SafetyReadOnly, SafetyChanges, SafetyDangerous:
	default:
		return fmt.Errorf("safety %q must be read-only, changes or dangerous", t.Safety)
	}
	switch t.Output {
	case "", "text", "json":
	default:
		return fmt.Errorf("output %q must be text or json", t.Output)
	}
	if t.Mode() == ModeDirect && t.Interpreter != "" {
		return errors.New("argv tasks run without an interpreter; remove \"interpreter\"")
	}
	return nil
}
