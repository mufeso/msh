// Package runbook runs a named list of steps on servers.
//
// A task is one remote action. A runbook is an operation made of several
// actions in a fixed order, such as "check disk, restart a service, verify
// it is active". Runbooks are defined under "runbooks:" in the
// configuration file.
package runbook

import (
	"example.com/msh/internal/config"
	"example.com/msh/internal/task"
)

// Step is one action of a runbook. It runs either a configured task or a
// shell command, never both.
type Step struct {
	Name    string            `json:"name" yaml:"name"`
	Task    string            `json:"task,omitempty" yaml:"task,omitempty"`
	Params  map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
	Args    []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Command string            `json:"command,omitempty" yaml:"command,omitempty"`

	// Interpreter runs Command (sh or bash; default sh).
	Interpreter string `json:"interpreter,omitempty" yaml:"interpreter,omitempty"`
	// AllowRedirection permits <, > and >> in Command.
	AllowRedirection bool `json:"allow_redirection,omitempty" yaml:"allow_redirection,omitempty"`
	// ReadOnly declares that Command changes nothing on the server.
	ReadOnly bool `json:"read_only,omitempty" yaml:"read_only,omitempty"`

	Retries         int              `json:"retries,omitempty" yaml:"retries,omitempty"`
	RetryDelay      *config.Duration `json:"retry_delay,omitempty" yaml:"retry_delay,omitempty"`
	Timeout         *config.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	ContinueOnError bool             `json:"continue_on_error,omitempty" yaml:"continue_on_error,omitempty"`
}

// Runbook is a named, ordered list of steps.
type Runbook struct {
	Name        string `json:"name" yaml:"-"` // the key in the "runbooks:" map
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Steps       []Step `json:"steps" yaml:"steps"`

	// Timeout limits the total time on each server.
	Timeout *config.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Arguments are the runbook's inputs, used in steps as {{name}}.
	Arguments []task.Argument `json:"arguments,omitempty" yaml:"arguments,omitempty"`

	// inputs validates and binds Arguments; it is set during validation.
	inputs *task.Task
}
