package task

import (
	"errors"
	"strings"
	"time"

	"example.com/msh/internal/shellquote"
)

// Invocation is a task ready to run on a server.
type Invocation struct {
	Command     string        // remote command line, parsed by the remote login shell
	Stdin       []byte        // script text for script mode, else nil
	Timeout     time.Duration // task-specific timeout; 0 = use the setting
	Interactive bool
}

// Render turns the task into an Invocation.
func (t *Task) Render() (Invocation, error) {
	if strings.Contains(t.Command, "{{") || strings.Contains(strings.Join(t.Argv, " "), "{{") {
		return Invocation{}, errors.New("task arguments are not supported yet")
	}
	inv := Invocation{Interactive: t.Interactive}
	if t.Timeout != nil {
		inv.Timeout = time.Duration(*t.Timeout)
	}
	in, err := LookupInterpreter(t.Interpreter)
	if err != nil {
		return Invocation{}, err
	}
	switch t.Mode() {
	case ModeDirect:
		inv.Command = shellquote.Join(t.Argv)
	case ModeShell:
		inv.Command = shellquote.Join([]string{in.Program, in.CommandFlag, t.Command})
	case ModeScript:
		inv.Command = shellquote.Join(append([]string{in.Program}, in.ScriptArgs...))
		inv.Stdin = []byte(t.Script)
	}
	return inv, nil
}
