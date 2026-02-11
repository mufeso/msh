package task

import (
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

// Render turns the task and its argument values into an Invocation.
//
// The remote login shell parses the final command line, so every word is
// quoted. How argument values enter depends on the mode:
//
//   - direct: placeholders are replaced inside individual argv words, and
//     each word is then quoted as a whole.
//   - shell:  each value is inserted as one single-quoted word into the
//     command template, which is then passed to "interpreter -c".
//   - script: values become positional parameters after "-s --"; the
//     script text is sent unchanged on standard input.
func (t *Task) Render(values Values) (Invocation, error) {
	inv := Invocation{Interactive: t.Interactive}
	if t.Timeout != nil {
		inv.Timeout = time.Duration(*t.Timeout)
	}
	identity := func(s string) string { return s }

	switch t.Mode() {
	case ModeDirect:
		words := make([]string, len(t.Argv))
		for i, w := range t.Argv {
			expanded, err := t.expand(w, values, identity)
			if err != nil {
				return Invocation{}, err
			}
			words[i] = expanded
		}
		inv.Command = shellquote.Join(words)
	case ModeShell:
		in, err := LookupInterpreter(t.Interpreter)
		if err != nil {
			return Invocation{}, err
		}
		body, err := t.expand(t.Command, values, shellquote.Quote)
		if err != nil {
			return Invocation{}, err
		}
		inv.Command = shellquote.Join([]string{in.Program, in.CommandFlag, body})
	case ModeScript:
		in, err := LookupInterpreter(t.Interpreter)
		if err != nil {
			return Invocation{}, err
		}
		words := append([]string{in.Program}, in.ScriptArgs...)
		for _, a := range t.Arguments {
			words = append(words, values[a.Name])
		}
		inv.Command = shellquote.Join(words)
		inv.Stdin = []byte(t.Script)
	}
	return inv, nil
}
