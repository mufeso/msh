# msh — My Parallel SSH

msh is a command-line SSH fleet-management tool for Linux servers. It
runs commands, reusable tasks, local scripts, multi-step runbooks and
interactive sessions on one or many servers, sequentially or in parallel,
directly or through a bastion, with verified host keys and clean CTRL+C
handling. It also collects facts, checks rpm and dpkg packages across the
fleet, and keeps a history of what it did.

A man page is in docs/man/msh.1. It is written in Go using the standard
library plus
`golang.org/x/crypto/ssh`, `golang.org/x/term` and `gopkg.in/yaml.v3`.

## Use it like ssh

```bash
msh user@server                       # open a shell, no setup needed
msh user@server df -h                 # run a command
msh -p 2222 -J jump@bastion server    # ssh flags and ~/.ssh/config work
msh --host "web01 web02" db01 uptime  # add servers: a,b or "a b" or @FILE
```

Keys from `ssh-add` work as they do with ssh. Otherwise msh asks for the
password, or reads `MSH_PASSWORD` in scripts.
See [using msh like ssh](docs/ssh-client.md).

## Quick start

```bash
make build && sudo install -m 0755 msh /usr/local/bin/msh
msh config init                  # ~/.config/msh/{config,inventory}.yaml
$EDITOR ~/.config/msh/inventory.yaml
msh config validate
msh hosts check                  # SSH connectivity and authentication
msh uptime --all
```

## Selecting servers

Every remote command accepts:

| Flag | Selects |
|------|---------|
| `--host web01,web02` | servers by name (repeatable) |
| `--group web` | members of a group (repeatable) |
| `--tag environment=production` | filters by tag (repeatable, all must match) |
| `--all` | every server |
| `--where 'os_family == "redhat"'` | filters by facts (see docs/facts.md) |

## Commands

```bash
# Inventory
msh hosts list [--group web] [--output json|csv|jsonl]
msh hosts check

# Arbitrary commands (run by the remote user's shell)
msh exec --host server01 --command "uname -a"
msh exec --group web --command "uptime"
msh exec --all --command "df -h" --mode parallel --workers 10 --ordered

# Tasks (see docs/tasks.md)
msh task list
msh task show message
msh task run message --host server01 --arg sshd
msh task run restart-service --group web --param service=nginx --yes
msh c message server01 sshd

# Local scripts: a file with many commands, streamed to the servers
msh script run --host server01 --file ./check.sh
msh script run --group web --file ./health-check.py --interpreter python3
msh script run --all --file examples/scripts/health-check.sh --arg nginx

# Runbooks: several steps, rolled out safely (see docs/runbooks.md)
msh runbook list
msh runbook show restart-web
msh runbook run restart-web --group web --dry-run
msh runbook run restart-web --group web --canary 1 --batch 25% --pause 30s --yes
msh runbook run restart-web --group web --yes --resume

# Facts (see docs/facts.md)
msh facts --all
msh facts --all --group-by os_version
msh facts --all --expect 'disk_root_pct < 90'

# Installed packages, rpm and dpkg (see docs/packages.md)
msh packages list kernel root@web01
msh packages check --all --package 'openssl>=3.0,!telnet'
msh packages check --all --file baseline.txt --only-problems
msh packages find openssl --all
msh packages compare --group web --baseline web01
msh packages updates --all

# History of runs (see docs/history.md)
msh history list --failed --since 24h
msh history show last
msh history diff 20261105T1015 last

# Built-in read-only shortcuts (msh help shortcuts)
msh ping --all
msh uname --all
msh uptime --group production
msh disk | memory | services | failed-units | selinux | kernel ...

# Logs
msh logs --host server01 --file /var/log/messages --lines 50
msh logs --group web --file /var/log/secure --follow

# Interactive sessions
msh shell --host server01
msh shell --host server01 --command "podman exec -it asdf /bin/bash"
msh shell --group web --tmux [--tmux-session fleet] [--tmux-layout tiled]

# Configuration and version
msh config init | show | validate
msh completion bash | zsh
msh version
msh --help
msh --version
```

## Execution

- `--mode parallel` (default) runs up to `--workers` servers at a time
  (default 5); `--mode sequential` runs one at a time.
- Results stream in completion order; `--ordered` prints them in selection
  order.
- `--command-timeout` limits each server, `--connect-timeout` each
  connection, `--timeout` the whole run.
- `--fail-fast` stops starting servers after the first problem.
- CTRL+C stops everything cleanly and exits 130.

## Output

`--output text|json|csv|jsonl`, server names on every line with several servers (`--blocks` for one block each),
`--report FILE` for a JSON record of the run. Diagnostics go to stderr
(`--verbose`, `--log-format json`). See docs/output.md.

## Bastions

```yaml
bastions:
  - name: production-bastion
    address: bastion.example.com
    user: jumpuser
servers:
  - name: app01
    address: 10.10.20.15
    bastion: production-bastion
```

`--bastion NAME` (or `none`) overrides the route. msh tunnels through the
bastion with direct-tcpip and never falls back to a direct connection.
See docs/bastion.md.

## Safety

Host keys are always verified, only key and agent authentication is used,
no passwords are stored, task arguments cannot inject shell syntax,
changing tasks need `--yes`, scripts never touch the remote disk, and
secrets never appear in output or logs. See docs/security.md.

## Exit codes

0 ok · 1 remote failure · 2 usage · 3 config · 4 connection · 5 timeout ·
6 transfer · 7 tmux · 8 file I/O · 70 internal · 130 canceled
(`msh help exit-codes`).

## Documentation

[like ssh](docs/ssh-client.md) · [install](docs/install.md) ·
[configuration](docs/configuration.md) ·
[commands](docs/commands.md) · [tasks](docs/tasks.md) ·
[runbooks](docs/runbooks.md) · [facts](docs/facts.md) ·
[packages](docs/packages.md) · [history](docs/history.md) ·
[scripts](docs/scripts.md) ·
[bastions](docs/bastion.md) · [interactive](docs/interactive.md) ·
[tmux](docs/tmux.md) · [output](docs/output.md) ·
[concurrency](docs/concurrency.md) · [shutdown](docs/shutdown.md) ·
[exit codes](docs/exit-codes.md) · [security](docs/security.md) ·
[learning path](docs/learning-path.md) · [examples](examples/)

## Development

```bash
./scripts/check.sh --race      # gofmt, vet, tests, race tests, build
go generate ./...              # regenerate the man page and command reference
make dist VERSION=2.0.0        # release binaries and SHA256SUMS
```

Tests need no servers: they start in-process SSH servers (including
bastions) on 127.0.0.1 with throwaway keys, and a fake tmux.
