package app

import (
	"bytes"
	"slices"
	"strings"

	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/interactive"
	"example.com/msh/internal/inventory"
)

const connectUsage = `Usage: msh [flags] [user@]host[:port] [command...]
       msh [flags] --host LIST [[user@]host [command...]]
       msh [flags] --group NAME|--tag K=V|--all [command...]
       msh connect [flags] [user@]host[:port] [command...]

Connects like ssh. Without a command it opens a shell. With one it runs the
command on the server, with raw output and the remote exit status. The
server does not need to be in the inventory; if it is, its settings apply.
If it is not, ~/.ssh/config is used as ssh would. Host keys are always
verified.

--host adds more servers: a,b or "a b" or @FILE. After it, the first word is
one more server if it looks like one (user@host, a name with a dot or port,
or a known name); otherwise it starts the command. --group, --tag and --all
take no destination: every word is the command. With several servers a
command runs on all of them in parallel, and without a command each server
opens in its own tmux pane.

Login uses keys from ssh-add first, then key files, then a password: the
MSH_PASSWORD environment variable, or a prompt on the terminal.

Examples:
  msh ops@server1
  msh root@server1:2222
  msh web01 df -h
  msh -t web01 top
  msh -p 2222 -l root -i ~/.ssh/key -J jump@bastion.example.com server1
  msh --host "web01 web02" root@db01 uptime
  msh --host web01,web02 root@db01 uptime
  msh --host web01,web02 ls -l /tmp
  msh --host @servers.txt root@db01 df -h
  msh --host web01,web02
  msh --group web uptime
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
	f.sel.register(fs)
	f.sel.registerWhere(fs)
	f.ro.register(fs)
	f.tmux.register(fs)
	fs.StringVar(&f.sel.forceUser, "l", "", "login `user` (like ssh -l)")
	fs.IntVar(&f.sel.forcePort, "p", 0, "`port` (like ssh -p)")
	fs.String("i", "", "private key `file` (like ssh -i)")
	fs.String("J", "", "connect through this `bastion`: a name or [user@]host[:port] (like ssh -J)")
	tty := fs.Bool("t", false, "use a terminal for a remote command (like ssh -t)")
	noTTY := fs.Bool("T", false, "never use a terminal (like ssh -T)")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}
	if f.sel.isEmpty() && fs.NArg() == 0 {
		c.errorf("usage: msh [user@]host [command...]")
		return exitcode.Usage
	}
	// Without a selection the first word is the destination, as in ssh.
	// After --host it may be one more server or the start of the command.
	if f.sel.isEmpty() || (f.sel.onlyHosts() && fs.NArg() > 0 && c.namesServer(&f, fs.Arg(0))) {
		if err := f.sel.hosts.Set(fs.Arg(0)); err != nil {
			c.errorf("%v", err)
			return exitcode.Usage
		}
		// A second round: like ssh, flags may follow the destination.
		// Parsing stops at the first other word, which starts the command.
		if code, ok := parseFlags(fs, fs.Args()[1:]); !ok {
			return code
		}
	}
	// As with ssh, the words are joined and the remote shell splits them.
	command := strings.Join(fs.Args(), " ")

	// Selecting replaces the entries by server names, and the session below
	// selects again, so it needs the user@ and :port of the original ones.
	hosts := slices.Clone(f.sel.hosts)
	st, servers, code := c.selectServers("connect", &f.opts, &f.sel)
	if st == nil {
		return code
	}
	f.sel.hosts = hosts
	if len(servers) > 1 && command != "" {
		return c.runRemote(remoteRun{
			opts: &f.opts, sel: &f.sel, ro: &f.ro, name: "connect",
			req: executor.Request{Command: command},
		})
	}
	if len(servers) > 1 {
		f.tmux.enabled = true
	}
	return c.interactiveSession(&f, interactive.Options{
		Command: command,
		NoTTY:   *noTTY || (command != "" && !*tty),
	})
}

// namesServer reports whether a word after --host is a server. A plain
// word like "ls" must be known by name: taking a command for a server would
// turn its options into msh's.
func (c *cli) namesServer(f *shellFlags, word string) bool {
	if strings.HasPrefix(word, "@") {
		return true
	}
	t, err := inventory.ParseTarget(word)
	if err != nil {
		return false
	}
	if t.User != "" || t.Port != 0 || strings.ContainsAny(t.Host, ".:") || c.userSSHConfig().Known(word) {
		return true
	}
	f.opts.inventoryOptional = true
	st, _ := c.load(&f.opts, true)
	if st == nil {
		return false
	}
	_, known := st.inv.Find(word)
	return known
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
	return len(word) < 4 || !looksLikeTypo(word) || c.userSSHConfig().Known(word)
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
