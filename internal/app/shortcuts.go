package app

import (
	"fmt"
	"strings"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
)

// shortcut is a built-in, read-only remote command available as
// "msh NAME". All shortcuts run through runRemote, so they share SSH,
// bastion, timeout, cancellation and output handling with "msh exec".
type shortcut struct {
	name    string
	summary string
	command string // remote command line
}

// shortcuts is the table of built-in shortcuts.
var shortcuts = []shortcut{
	{name: "ping", summary: "Check SSH connectivity and authentication", command: "true"},
	{name: "uname", summary: "Show system information (uname -a)", command: "uname -a"},
	{name: "kernel", summary: "Show the kernel release (uname -r)", command: "uname -r"},
	{name: "uptime", summary: "Show uptime and load averages", command: "uptime"},
	{name: "disk", summary: "Show filesystem usage (df -hP)", command: "df -hP"},
	{name: "memory", summary: "Show memory usage in MiB (free -m)", command: "free -m"},
	{name: "services", summary: "List running systemd services", command: "systemctl list-units --type=service --state=running --no-pager --no-legend"},
	{name: "failed-units", summary: "List failed systemd units", command: "systemctl --failed --no-pager --no-legend"},
	{name: "selinux", summary: "Show the SELinux mode (getenforce)", command: "if command -v getenforce >/dev/null 2>&1; then getenforce; else echo 'SELinux tools are not installed'; fi"},
}

// shortcutCommands turns the table into CLI commands.
func shortcutCommands() []*command {
	cmds := make([]*command, 0, len(shortcuts))
	for _, sc := range shortcuts {
		usage := fmt.Sprintf(`Usage: msh %s ([user@]host | --host LIST | --group NAME | --tag K=V | --all) [flags]

%s.
Remote command: %s
`, sc.name, sc.summary, sc.command)
		cmds = append(cmds, &command{
			name:     sc.name,
			summary:  sc.summary,
			usage:    usage,
			shortcut: true,
			// sc is a fresh variable in each iteration (Go 1.22+), so every
			// closure captures its own shortcut.
			run: func(c *cli, args []string) exitcode.Code {
				return c.runShortcut(sc, usage, args, false)
			},
		})
	}
	return cmds
}

// runShortcut parses the usual flags and runs the shortcut's command.
// With defaultAll, an empty selection means all servers.
func (c *cli) runShortcut(sc shortcut, usage string, args []string, defaultAll bool) exitcode.Code {
	var opts commonOptions
	var sel selectionOptions
	var ro remoteOptions
	fs := c.newFlagSet(sc.name, usage)
	opts.register(fs)
	sel.register(fs)
	sel.registerWhere(fs)
	ro.register(fs)
	if _, code, ok := parseRemote(fs, &sel, args, 0); !ok {
		return code
	}
	if defaultAll && sel.isEmpty() {
		sel.all = true
	}
	return c.runRemote(remoteRun{
		opts: &opts, sel: &sel, ro: &ro, name: sc.name,
		req: executor.Request{Command: sc.command},
	})
}

// lookupShortcut returns the shortcut with the given name.
func lookupShortcut(name string) shortcut {
	for _, sc := range shortcuts {
		if sc.name == name {
			return sc
		}
	}
	panic("unknown shortcut " + name) // a programming error, caught by tests
}

// shortcutsHelp lists every shortcut and its remote command.
func shortcutsHelp() string {
	var b strings.Builder
	b.WriteString("Built-in shortcuts. Each runs one read-only remote command and accepts\n")
	b.WriteString("the same selection, output and timeout flags as \"msh exec\".\n\n")
	for _, sc := range shortcuts {
		fmt.Fprintf(&b, "  %-13s %s\n  %-13s $ %s\n", sc.name, sc.summary, "", sc.command)
	}
	return b.String()
}
