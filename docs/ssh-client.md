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

`--host` is only needed for more than one server.

```bash
msh --host web01,web02 uptime             # comma-separated
msh --host "web01 web02 root@db01" uptime # space-separated
msh --host @servers.txt df -h             # @ reads a file
msh --host a --host @more.txt uptime      # repeat and mix them
msh --group web uptime
```

`--host a` is the server `a`; `--host @a` is the file `a`. A file has server
names separated by lines, spaces or commas, and `#` starts a comment.

With a command, msh runs it on all servers in parallel and prints its usual
report. Without a command, each server opens in its own tmux pane. The same
`--host` forms work with every msh command, for example
`msh exec --host @servers.txt --command uptime`.

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
- Authentication uses keys and ssh-agent. There is no password prompt.
- A ProxyJump with several hops is an error; msh never connects directly
  instead.
- In `~/.ssh/config`, Match blocks and options other than the five above are
  ignored.
