package app

import (
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"example.com/msh/internal/exitcode"
)

const completionUsage = `Usage: msh completion bash|zsh

Prints a shell completion script. Load it with:

  source <(msh completion bash)     # in ~/.bashrc
  source <(msh completion zsh)      # in ~/.zshrc, after compinit
`

const bashCompletion = `_msh() {
  local IFS=$'\n'
  COMPREPLY=($(msh __complete "${COMP_WORDS[@]:1:COMP_CWORD}" 2>/dev/null))
}
complete -o default -F _msh msh
`

const zshCompletion = `_msh() {
  local -a candidates
  candidates=("${(@f)$(msh __complete "${(@)words[2,CURRENT]}" 2>/dev/null)}")
  compadd -a candidates
}
compdef _msh msh
`

func completionCommand() *command {
	return &command{
		name:    "completion",
		summary: "Print a shell completion script",
		usage:   completionUsage,
		run: func(c *cli, args []string) exitcode.Code {
			if len(args) == 1 && args[0] == "bash" {
				fmt.Fprint(c.stdout, bashCompletion)
				return exitcode.OK
			}
			if len(args) == 1 && args[0] == "zsh" {
				fmt.Fprint(c.stdout, zshCompletion)
				return exitcode.OK
			}
			fmt.Fprint(c.stderr, completionUsage)
			return exitcode.Usage
		},
	}
}

// completeCommand is the hidden helper the scripts call. Its arguments
// are the words after "msh"; the last one is being typed.
func completeCommand() *command {
	return &command{
		name:   "__complete",
		hidden: true,
		run: func(c *cli, args []string) exitcode.Code {
			for _, s := range c.complete(args) {
				fmt.Fprintln(c.stdout, s)
			}
			return exitcode.OK
		},
	}
}

// subcommands returns the subcommands of a command.
func subcommands(name string) []string {
	switch name {
	case "completion":
		return []string{"bash", "zsh"}
	case "config":
		return []string{"init", "show", "validate"}
	case "history":
		return []string{"list", "show", "diff", "prune"}
	case "hosts":
		return []string{"list", "check"}
	case "packages":
		return []string{"list", "check", "find", "compare", "updates"}
	case "runbook", "task":
		return []string{"list", "show", "run"}
	case "script":
		return []string{"run"}
	default:
		return nil
	}
}

// complete returns the candidates for the last word.
func (c *cli) complete(words []string) []string {
	if len(words) == 0 {
		words = []string{""}
	}
	current := words[len(words)-1]
	if strings.HasPrefix(current, "-") {
		return withPrefix(commonFlags, current)
	}
	var candidates []string
	if len(words) >= 2 {
		switch words[len(words)-2] {
		case "--host":
			return withPrefix(c.inventoryNames(servers), current)
		case "--group":
			return withPrefix(c.inventoryNames(groups), current)
		case "--bastion":
			return withPrefix(append(c.inventoryNames(bastions), "none"), current)
		}
	}
	switch {
	case len(words) == 2 && words[0] == "c":
		return withPrefix(c.configNames(false), current)
	case len(words) == 3 && words[0] == "c":
		return withPrefix(c.inventoryNames(servers), current)
	case len(words) == 3 && words[0] == "task" && (words[1] == "run" || words[1] == "show"):
		return withPrefix(c.configNames(false), current)
	case len(words) == 3 && words[0] == "runbook" && (words[1] == "run" || words[1] == "show"):
		return withPrefix(c.configNames(true), current)
	}
	switch len(words) {
	case 1:
		for _, cmd := range commands() {
			if !cmd.hidden {
				candidates = append(candidates, cmd.name)
			}
		}
	case 2:
		candidates = subcommands(words[0])
		if words[0] == "help" {
			candidates = commandNames()
		}
	}
	return withPrefix(candidates, current)
}

// withPrefix keeps the candidates starting with prefix.
func withPrefix(candidates []string, prefix string) []string {
	var out []string
	for _, s := range candidates {
		if strings.HasPrefix(s, prefix) {
			out = append(out, s)
		}
	}
	return out
}

// commonFlags are offered for words starting with "-".
var commonFlags = []string{
	"--all", "--bastion", "--command", "--config", "--group", "--help", "--host",
	"--inventory", "--mode", "--blocks", "--ordered", "--output", "--prefix", "--tag",
	"--timeout", "--verbose", "--where", "--workers",
}

// Name kinds for inventoryNames.
const (
	servers = iota
	groups
	bastions
)

// quietSetup loads the configuration without printing anything: errors
// must not disturb the shell during completion.
func (c *cli) quietSetup(needInventory bool) *setup {
	var opts commonOptions
	fs := flag.NewFlagSet("complete", flag.ContinueOnError)
	opts.register(fs)
	quiet := *c
	quiet.stderr = io.Discard
	st, _ := quiet.load(&opts, needInventory)
	return st
}

// inventoryNames returns server, group or bastion names.
func (c *cli) inventoryNames(kind int) []string {
	st := c.quietSetup(true)
	if st == nil {
		return nil
	}
	var out []string
	switch kind {
	case bastions:
		for _, b := range st.inv.Bastions {
			out = append(out, b.Name)
		}
	case groups:
		for _, s := range st.inv.Servers {
			for _, g := range s.Groups {
				if !slices.Contains(out, g) {
					out = append(out, g)
				}
			}
		}
		slices.Sort(out)
	default:
		out = st.inv.Names()
	}
	return out
}

// configNames returns task names, or runbook names.
func (c *cli) configNames(runbooks bool) []string {
	st := c.quietSetup(false)
	if st == nil {
		return nil
	}
	if runbooks {
		return st.runbooks.Names()
	}
	return st.tasks.Names()
}
