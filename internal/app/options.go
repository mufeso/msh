package app

import (
	"flag"
	"fmt"
	"time"

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
	"inventory":       config.KeyInventory,
	"user":            config.KeyUser,
	"port":            config.KeyPort,
	"identity":        config.KeyIdentityFile,
	"known-hosts":     config.KeyKnownHostsFile,
	"output":          config.KeyOutput,
	"connect-timeout": config.KeyConnectTimeout,
	"command-timeout": config.KeyCommandTimeout,
	"timeout":         config.KeyTimeout,
	"mode":            config.KeyMode,
	"workers":         config.KeyWorkers,
	"ordered":         config.KeyOrdered,
	"bastion":         config.KeyBastion,
}

// register adds the common flags to fs. The defaults shown in help come
// from config.Defaults, the single source of truth.
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
	fs.Duration("connect-timeout", d.ConnectTimeout, "TCP connect and SSH handshake `timeout` per server")
	fs.Duration("command-timeout", d.CommandTimeout, "remote command `timeout` per server (0 = none)")
	fs.Duration("timeout", d.Timeout, "global `timeout` for the whole run (0 = none)")
	fs.String("mode", d.Mode, "execution `mode`: sequential or parallel")
	fs.Int("workers", d.Workers, "maximum concurrent `servers` in parallel mode")
	fs.Bool("ordered", d.Ordered, "print results in selection order instead of completion order")
	fs.String("bastion", "", "connect through this bastion `name` (\"none\" = direct)")
}

// settingsLayer builds a configuration layer from the flags that were set
// on the command line.
func (o *commonOptions) settingsLayer() config.Layer {
	var l config.Layer
	o.fs.Visit(func(f *flag.Flag) {
		v := f.Value.String()
		// Only standard flag types implement flag.Getter, so the type
		// assertion happens lazily, inside the cases that need it.
		get := func() any { return f.Value.(flag.Getter).Get() }
		switch settingsFlags[f.Name] {
		case config.KeyInventory:
			l.Inventory = &v
		case config.KeyUser:
			l.User = &v
		case config.KeyPort:
			p := get().(int)
			l.Port = &p
		case config.KeyIdentityFile:
			l.IdentityFile = &v
		case config.KeyKnownHostsFile:
			l.KnownHostsFile = &v
		case config.KeyOutput:
			l.Output = &v
		case config.KeyConnectTimeout:
			l.ConnectTimeout = durationPtr(get().(time.Duration))
		case config.KeyCommandTimeout:
			l.CommandTimeout = durationPtr(get().(time.Duration))
		case config.KeyTimeout:
			l.Timeout = durationPtr(get().(time.Duration))
		case config.KeyMode:
			l.Mode = &v
		case config.KeyWorkers:
			n := get().(int)
			l.Workers = &n
		case config.KeyOrdered:
			b := get().(bool)
			l.Ordered = &b
		case config.KeyBastion:
			l.Bastion = &v
		}
	})
	return l
}

func durationPtr(d time.Duration) *config.Duration {
	v := config.Duration(d)
	return &v
}

// verbosef prints a diagnostic to stderr when verbose is enabled.
// Diagnostics never go to stdout, so they cannot corrupt JSON output.
func (c *cli) verbosef(opts *commonOptions, format string, a ...any) {
	if opts.verbose {
		fmt.Fprintf(c.stderr, "msh: "+format+"\n", a...)
	}
}
