package app

import (
	"fmt"
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
	var candidates []string
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
