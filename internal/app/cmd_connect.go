package app

import (
	"bytes"
	"slices"
	"strings"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
)

const connectUsage = `Usage: msh [flags] [user@]host[:port] [command...]
       msh connect [flags] [user@]host[:port] [command...]

Connects like ssh. Without a command it opens a shell. With one it runs the
command on the server, with raw output and the remote exit status. The
server does not need to be in the inventory; if it is, its settings apply.
Host keys are always verified.

Examples:
  msh ops@server1
  msh root@server1:2222
  msh web01 df -h
  msh -t web01 top
  msh -p 2222 -l root -i ~/.ssh/key -J jump@bastion.example.com server1
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
	fs.StringVar(&f.sel.forceUser, "l", "", "login `user` (like ssh -l)")
	fs.IntVar(&f.sel.forcePort, "p", 0, "`port` (like ssh -p)")
	fs.String("i", "", "private key `file` (like ssh -i)")
	fs.String("J", "", "connect through this `bastion`: a name or [user@]host[:port] (like ssh -J)")
	tty := fs.Bool("t", false, "use a terminal for a remote command (like ssh -t)")
	noTTY := fs.Bool("T", false, "never use a terminal (like ssh -T)")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() == 0 {
		c.errorf("usage: msh [user@]host [command...]")
		return exitcode.Usage
	}
	if err := f.sel.hosts.Set(fs.Arg(0)); err != nil {
		c.errorf("%v", err)
		return exitcode.Usage
	}
	// A second round: like ssh, flags may follow the destination. Parsing
	// stops at the first other word, which starts the remote command.
	if code, ok := parseFlags(fs, fs.Args()[1:]); !ok {
		return code
	}
	// As with ssh, the words are joined and the remote shell splits them.
	command := strings.Join(fs.Args(), " ")
	return c.interactiveSession(&f, interactive.Options{
		Command: command,
		NoTTY:   *noTTY || (command != "" && !*tty),
	})
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
