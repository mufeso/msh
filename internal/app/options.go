package app

import (
	"flag"
	"fmt"

	"example.com/msh/internal/config"
	"example.com/msh/internal/output"
)

// commonOptions are flags accepted by most commands.
type commonOptions struct {
	configPath string
	verbose    bool

	// fs is kept so that settingsLayer can see which flags were set.
	fs *flag.FlagSet

	// format is filled in from the resolved settings.
	format output.Format
}

// settingsFlags maps settings flags to configuration keys.
var settingsFlags = map[string]string{
	"inventory":   config.KeyInventory,
	"user":        config.KeyUser,
	"port":        config.KeyPort,
	"identity":    config.KeyIdentityFile,
	"known-hosts": config.KeyKnownHostsFile,
	"output":      config.KeyOutput,
}

// register adds the common flags to fs.
//
// The settings flags are registered without destination variables we read
// directly: their defaults are only shown in help. settingsLayer later
// reads the values of the flags the user actually typed.
func (o *commonOptions) register(fs *flag.FlagSet) {
	o.fs = fs
	d := config.Defaults()
	fs.StringVar(&o.configPath, "config", "", "configuration `file`")
	fs.BoolVar(&o.verbose, "verbose", false, "print diagnostics to stderr")
	fs.String("inventory", "", "inventory `file`")
	fs.String("user", "", "default SSH `user` for servers without one")
	fs.Int("port", d.Port, "default SSH `port` for servers without one")
	fs.String("identity", "", "private key `file`")
	fs.String("known-hosts", "", "known_hosts `file`")
	fs.String("output", d.Output, "output `format`: text or json")
}

// settingsLayer builds a configuration layer from the flags that were set
// on the command line. Flags left at their defaults are not included, so
// they cannot override a configuration file.
func (o *commonOptions) settingsLayer() config.Layer {
	var l config.Layer
	o.fs.Visit(func(f *flag.Flag) {
		// v is a new variable on every call, so each pointer below points
		// at its own value.
		v := f.Value.String()
		switch settingsFlags[f.Name] {
		case config.KeyInventory:
			l.Inventory = &v
		case config.KeyUser:
			l.User = &v
		case config.KeyPort:
			// flag.Getter exposes the typed value; the type assertions
			// recover the int stored by fs.Int.
			p := f.Value.(flag.Getter).Get().(int)
			l.Port = &p
		case config.KeyIdentityFile:
			l.IdentityFile = &v
		case config.KeyKnownHostsFile:
			l.KnownHostsFile = &v
		case config.KeyOutput:
			l.Output = &v
		}
	})
	return l
}

// verbosef prints a diagnostic to stderr when verbose is enabled.
// Diagnostics never go to stdout, so they cannot corrupt JSON output.
func (c *cli) verbosef(opts *commonOptions, format string, a ...any) {
	if opts.verbose {
		fmt.Fprintf(c.stderr, "msh: "+format+"\n", a...)
	}
}
