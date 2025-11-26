package app

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"example.com/msh/internal/exitcode"
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
