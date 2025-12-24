package app

import "example.com/msh/internal/exitcode"

// command describes one msh subcommand.
type command struct {
	name    string
	summary string // one line, shown in the command list
	usage   string // full help text, shown by "msh help <name>"

	// run is a function value: each command stores the function that
	// implements it. It receives the arguments after the command name.
	run func(c *cli, args []string) exitcode.Code
}

// commands returns every command in the order shown by --help.
//
// It is a function rather than a package-level variable so that commands
// such as help can call it without creating an initialization cycle.
func commands() []*command {
	return []*command{
		configCommand(),
		helpCommand(),
		hostsCommand(),
		versionCommand(),
	}
}

// lookup finds a command by name.
func lookup(name string) (*command, bool) {
	for _, cmd := range commands() {
		if cmd.name == name {
			return cmd, true
		}
	}
	return nil, false
}
