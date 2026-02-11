package task

import "fmt"

// Interpreter describes how to start a language interpreter remotely.
type Interpreter struct {
	Program     string   // executable name
	CommandFlag string   // flag that takes a program text: -c, -e
	ScriptArgs  []string // arguments that make it read the program from stdin
	POSIXShell  bool     // true for shells that understand POSIX quoting
}

// interpreters are the supported interpreters. The program reads the
// script from standard input, so nothing is written to the remote disk.
var interpreters = map[string]Interpreter{
	"sh":      {Program: "sh", CommandFlag: "-c", ScriptArgs: []string{"-s", "--"}, POSIXShell: true},
	"bash":    {Program: "bash", CommandFlag: "-c", ScriptArgs: []string{"-s", "--"}, POSIXShell: true},
	"python3": {Program: "python3", CommandFlag: "-c", ScriptArgs: []string{"-"}},
	"perl":    {Program: "perl", CommandFlag: "-e", ScriptArgs: []string{"-"}},
}

// LookupInterpreter returns the interpreter for name ("" means sh).
func LookupInterpreter(name string) (Interpreter, error) {
	if name == "" {
		name = "sh"
	}
	in, ok := interpreters[name]
	if !ok {
		return Interpreter{}, fmt.Errorf("unknown interpreter %q (supported: sh, bash, python3, perl)", name)
	}
	return in, nil
}
