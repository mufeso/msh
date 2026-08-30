package app

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
)

// newFlagSet creates a flag set for one command.
//
// flag.ContinueOnError makes Parse return an error instead of calling
// os.Exit, which keeps commands testable. Errors and usage go to stderr.
func (c *cli) newFlagSet(name, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet("msh "+name, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	fs.Usage = func() {
		printFlagUsage(c.stderr, fs, usage)
	}
	return fs
}

// printFlagUsage prints the usage text followed by the flag defaults.
func printFlagUsage(w io.Writer, fs *flag.FlagSet, usage string) {
	fmt.Fprint(w, usage)
	hasFlags := false
	fs.VisitAll(func(*flag.Flag) { hasFlags = true })
	if hasFlags {
		fmt.Fprintln(w, "\nFlags:")
		fs.PrintDefaults()
	}
}

// parseFlags parses args. When ok is false the command must return code
// immediately: either help was requested (exit 0) or the flags were
// invalid (exit 2). The flag package has already printed the message.
func parseFlags(fs *flag.FlagSet, args []string) (code exitcode.Code, ok bool) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitcode.OK, false
		}
		return exitcode.Usage, false
	}
	return exitcode.OK, true
}

// parseInterspersed parses flags that may appear before, between or after
// positional arguments ("msh task show NAME --config FILE"). The standard
// flag package stops at the first non-flag argument, so this loops:
// parse, take one positional argument, parse the rest again. Everything
// after a "--" argument is positional.
func parseInterspersed(fs *flag.FlagSet, args []string) (positional []string, code exitcode.Code, ok bool) {
	for {
		if code, ok := parseFlags(fs, args); !ok {
			return nil, code, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, exitcode.OK, true
		}
		// The flag package consumes a "--" terminator itself; detect it by
		// comparing with the input.
		if len(args) > len(rest) && args[len(args)-len(rest)-1] == "--" {
			return append(positional, rest...), exitcode.OK, true
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

// parseRemote parses the arguments of a command that connects to servers.
// A user@host word is always a destination. Other words first fill the
// keep words the command uses itself, which it gets back, and the rest are
// destinations too. A negative keep gives every plain word to the command.
func parseRemote(fs *flag.FlagSet, sel *selectionOptions, args []string, keep int) (words []string, code exitcode.Code, ok bool) {
	positional, code, ok := parseInterspersed(fs, args)
	if !ok {
		return nil, code, false
	}
	for _, word := range positional {
		t, err := inventory.ParseTarget(word)
		if (err != nil || t.User == "") && (keep < 0 || len(words) < keep) {
			words = append(words, word)
			continue
		}
		if err := sel.hosts.Set(word); err != nil {
			fmt.Fprintf(fs.Output(), "msh: %v\n", err)
			return nil, exitcode.Usage, false
		}
	}
	return words, exitcode.OK, true
}
