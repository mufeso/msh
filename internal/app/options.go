package app

import (
	"flag"
	"fmt"
)

// commonOptions are flags accepted by most commands. They are kept in one
// struct so every command registers and interprets them the same way.
type commonOptions struct {
	configPath    string
	inventoryPath string
	output        string
	verbose       bool
}

// register adds the common flags to fs. The flag package stores parsed
// values directly into the struct fields through the pointers we pass.
func (o *commonOptions) register(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "configuration `file`")
	fs.StringVar(&o.inventoryPath, "inventory", "", "inventory `file`")
	fs.StringVar(&o.output, "output", "text", "output `format`: text or json")
	fs.BoolVar(&o.verbose, "verbose", false, "print diagnostics to stderr")
}

// validate checks the values after parsing.
func (o *commonOptions) validate() error {
	switch o.output {
	case "text", "json":
		return nil
	default:
		return fmt.Errorf("invalid --output %q: must be text or json", o.output)
	}
}

// verbosef prints a diagnostic to stderr when verbose is enabled.
// Diagnostics never go to stdout, so they cannot corrupt JSON output.
func (c *cli) verbosef(opts *commonOptions, format string, a ...any) {
	if opts.verbose {
		fmt.Fprintf(c.stderr, "msh: "+format+"\n", a...)
	}
}
