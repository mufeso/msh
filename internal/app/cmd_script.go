package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/shellquote"
	"example.com/msh/internal/task"
)

const scriptUsage = `Usage: msh script run (--host NAME | --group NAME | --tag K=V | --all) --file PATH [--interpreter NAME] [--arg VALUE]... [flags]

Streams a local script to an interpreter on each selected server through
SSH standard input ("bash -s --", "sh -s --", "python3 -", "perl -").
The script is never copied to the server's disk. Supported interpreters:
sh, bash, python3, perl. Without --interpreter the script's shebang line
decides, else sh.
`

// maxScriptSize bounds what is read into memory and sent to every server.
const maxScriptSize = 16 << 20

func scriptCommand() *command {
	return &command{
		name:    "script",
		summary: "Run a local script on servers without copying it",
		usage:   scriptUsage,
		run:     runScript,
	}
}

func runScript(c *cli, args []string) exitcode.Code {
	if len(args) == 0 || args[0] != "run" {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
			fmt.Fprint(c.stdout, scriptUsage)
			return exitcode.OK
		}
		fmt.Fprint(c.stderr, scriptUsage)
		return exitcode.Usage
	}
	var opts commonOptions
	var sel selectionOptions
	var ro remoteOptions
	fs := c.newFlagSet("script run", scriptUsage)
	opts.register(fs)
	sel.register(fs)
	sel.registerWhere(fs)
	ro.register(fs)
	file := fs.String("file", "", "local script `path`")
	interpName := fs.String("interpreter", "", "remote `interpreter`: sh, bash, python3 or perl")
	var scriptArgs []string
	fs.Func("arg", "script argument `value` (repeatable): \"$1\", sys.argv[1], ...", func(v string) error {
		scriptArgs = append(scriptArgs, v)
		return nil
	})
	if code, ok := parseFlags(fs, args[1:]); !ok {
		return code
	}
	if *file == "" {
		c.errorf("script run: --file is required")
		return exitcode.Usage
	}
	script, err := readScript(*file)
	if err != nil {
		c.errorf("script run: %v", err)
		return exitcode.IO
	}
	name := *interpName
	if name == "" {
		name = shebangInterpreter(script)
	}
	interp, err := task.LookupInterpreter(name)
	if err != nil {
		c.errorf("script run: %v; choose one with --interpreter", err)
		return exitcode.Usage
	}
	words := append([]string{interp.Program}, interp.ScriptArgs...)
	words = append(words, scriptArgs...)
	c.verbosef(&opts, "streaming %s (%d bytes) to %q", *file, len(script), shellquote.Join(words))
	return c.runRemote(remoteRun{
		opts: &opts, sel: &sel, ro: &ro, name: "script run",
		req: executor.Request{Command: shellquote.Join(words), Stdin: script},
	})
}

// readScript reads a regular file of at most maxScriptSize bytes.
func readScript(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	// Read one byte more than allowed to detect oversized files even if
	// the file grows while we read it.
	data, err := io.ReadAll(io.LimitReader(f, maxScriptSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxScriptSize {
		return nil, fmt.Errorf("%s is larger than %d MiB", path, maxScriptSize>>20)
	}
	if len(data) == 0 {
		return nil, errors.New(path + " is empty")
	}
	return data, nil
}

// shebangInterpreter returns the interpreter named by a "#!" line:
// "#!/bin/bash" -> bash, "#!/usr/bin/env python3" -> python3. It returns
// "" when there is no shebang.
func shebangInterpreter(script []byte) string {
	first, _, _ := strings.Cut(string(script), "\n")
	rest, ok := strings.CutPrefix(first, "#!")
	if !ok {
		return ""
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	prog := path.Base(fields[0])
	if prog == "env" {
		for _, f := range fields[1:] {
			if !strings.HasPrefix(f, "-") {
				return path.Base(f)
			}
		}
		return ""
	}
	return prog
}
