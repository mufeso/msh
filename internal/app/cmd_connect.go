package app

import (
	"bytes"
	"slices"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
)

const connectUsage = `Usage: msh [flags] [user@]host[:port]
       msh connect [flags] [user@]host[:port]

Connects like ssh: opens a shell on the server. The server does not need
to be in the inventory; if it is, its settings apply. Host keys are always
verified.

Examples:
  msh ops@server1
  msh root@server1:2222
  msh web01
`

func connectCommand() *command {
	return &command{
		name:    "connect",
		summary: "Connect like ssh (msh [user@]host also works)",
		usage:   connectUsage,
		run:     runConnect,
	}
}

func runConnect(c *cli, args []string) exitcode.Code {
	var f shellFlags
	fs := c.newFlagSet("connect", connectUsage)
	f.opts.register(fs)
	f.ro.register(fs)
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() != 1 {
		c.errorf("usage: msh [user@]host")
		return exitcode.Usage
	}
	if err := f.sel.hosts.Set(fs.Arg(0)); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	return c.interactiveSession(&f, interactive.Options{})
}

// isDestination decides what an unknown first word means: flags and
// words with @ . or : are for connecting, as is any inventory name. A word
// that is close to a command name is more likely a typo.
func (c *cli) isDestination(word string) bool {
	if strings.HasPrefix(word, "-") || strings.ContainsAny(word, "@.:") {
		return true
	}
	if st := c.quietSetup(true); st != nil {
		if _, ok := st.inv.Find(word); ok {
			return true
		}
	}
	return len(word) < 4 || !looksLikeTypo(word)
}

// looksLikeTypo is stricter than suggest, because a wrong yes here keeps a
// server from being reached: one letter off, or the same letters swapped.
func looksLikeTypo(word string) bool {
	for _, name := range commandNames() {
		if levenshtein(word, name) <= 1 || sameLetters(word, name) {
			return true
		}
	}
	return false
}

func sameLetters(a, b string) bool {
	x, y := []byte(a), []byte(b)
	slices.Sort(x)
	slices.Sort(y)
	return bytes.Equal(x, y)
}
