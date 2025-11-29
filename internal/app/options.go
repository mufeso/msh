package app

import (
	"flag"
	"fmt"

	"example.com/msh/internal/output"
)

// commonOptions are flags accepted by most commands.
type commonOptions struct {
	configPath    string
	inventoryPath string
	outputName    string
	verbose       bool

	// format is set by validate from outputName.
	format output.Format
}

// register adds the common flags to fs.
func (o *commonOptions) register(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "configuration `file`")
	fs.StringVar(&o.inventoryPath, "inventory", "", "inventory `file`")
	fs.StringVar(&o.outputName, "output", "text", "output `format`: text or json")
	fs.BoolVar(&o.verbose, "verbose", false, "print diagnostics to stderr")
}

// validate checks the values after parsing and converts them to typed
// values.
func (o *commonOptions) validate() error {
	f, err := output.ParseFormat(o.outputName)
	if err != nil {
		return fmt.Errorf("--output: %w", err)
	}
	o.format = f
	return nil
}

// verbosef prints a diagnostic to stderr when verbose is enabled.
// Diagnostics never go to stdout, so they cannot corrupt JSON output.
func (c *cli) verbosef(opts *commonOptions, format string, a ...any) {
	if opts.verbose {
		fmt.Fprintf(c.stderr, "msh: "+format+"\n", a...)
	}
}
