# Using msh like ssh

msh connects like an ssh client. No inventory or setup is needed.

```bash
msh user@server                # open a shell
msh server                     # same, with your local user name
msh root@server:2222           # another port
msh server df -h               # run a command; raw output, remote exit status
msh server cat /etc/hostname | wc -l
msh -t server top              # a command that needs a terminal
```

The first word is a msh command if it is one (`exec`, `shell`, ...).
Anything else is a destination. For a server named like a command, or
like a typo of one, use `msh connect NAME`.

## Destinations

`host`, `user@host`, `user@host:port`, `[2001:db8::1]:22` and
`ssh://user@host:port` all work. A name from the inventory or an alias from
`~/.ssh/config` works too.

## Flags

| Flag | Like ssh | Meaning |
|------|----------|---------|
| `-l USER` | `-l` | login user, even over the inventory |
| `-p PORT` | `-p` | port, even over the inventory |
| `-i FILE` | `-i` | private key |
| `-J [user@]host[:port]` | `-J` | connect through a jump host: a bastion name, an alias or a target |
| `-t` | `-t` | use a terminal for a remote command |
| `-T` | `-T` | never use a terminal |

Flags can go before or after the destination. Everything after the first
word of the remote command belongs to the command.

## Several servers

`--host` adds more servers to the destination. The first word after the flags
is always a destination, as in ssh, and the words after it are the command.

```bash
msh --host "web01 web02" root@db01 uptime   # space-separated
msh --host web01,web02 root@db01 uptime     # comma-separated
msh --host "web01, web02" root@db01 uptime  # both
msh --host @a root@db01 uptime              # @ reads the file a
msh --host web01,web02                      # no destination, no command: panes
```

Each of the first four runs `uptime` on three servers. Be careful with
`msh --host a,b uptime`: here `uptime` is a destination, not the command.

`--host a` is the server `a`; `--host @a` is the file `a`. A file has server
names separated by lines, spaces or commas, and `#` starts a comment.

`--group`, `--tag` and `--all` have no destination, so every word is the
command:

```bash
msh --group web uptime
msh --all df -h
```

With a command, msh runs it on all servers in parallel and prints its usual
report. Without a command, each server opens in its own tmux pane. To run on
exactly the listed servers, use the explicit form, which takes the same
`--host` values: `msh exec --host @a --command uptime`.

## Other commands

Every remote command takes destinations the same way. A `user@host` word is
always a destination. Other words first fill what the command needs, and the
rest are destinations too.

```bash
msh packages list tmux root@db01         # tmux is the pattern
msh packages check tmux 'nginx>=1.20' root@db01
msh task run restart-nginx root@db01     # the first word is the task
msh facts root@db01 web01
msh uptime web01 web02
msh exec root@db01 --command uptime
```

Where a word has a meaning of its own, write `user@host` or use `--host`:
`msh packages list web01` looks for a package called web01.

## Logging in

msh tries, in this order:

1. keys in ssh-agent: `ssh-add ~/.ssh/key`, then just `msh user@server`
2. key files: `-i FILE`, or `~/.ssh/id_ed25519`, `id_ecdsa` and `id_rsa`
3. a password: `MSH_PASSWORD` if it is set, otherwise a prompt on the terminal

```bash
read -rs MSH_PASSWORD && export MSH_PASSWORD   # not visible, not in history
msh --host @a root@db01 uptime
unset MSH_PASSWORD
```

Don't write `MSH_PASSWORD=... msh` on the command line: it ends up in your
shell history. Without a terminal and without the variable nothing is asked,
so cron jobs fail with a clear error instead of hanging. A password is only
sent after the host key was verified. See [security](security.md).

## What decides the settings

For a server that is not in the inventory, highest first:

1. `user@`, `:port`, `-l`, `-p`, `--user`, `--port`, `-J`, `MSH_*` variables
2. `~/.ssh/config`: HostName, User, Port, IdentityFile, ProxyJump
3. msh's configuration and defaults

Servers in the inventory keep their settings and do not read
`~/.ssh/config`. `user@`, `:port`, `-l` and `-p` override them.

## Differences from ssh

- Host keys are always verified, and an unknown server is refused instead of
  asking yes or no. Add it first: `ssh-keyscan host >> ~/.ssh/known_hosts`
  after checking the key, or connect once with ssh.
- A wrong password is not asked again: run the command again. With several
  servers, one prompt per login user serves them all.
- A ProxyJump with several hops is an error; msh never connects directly
  instead.
- In `~/.ssh/config`, Match blocks and options other than the five above are
  ignored.
