# Changelog

## 2.4.0

- Changed: with several servers, text output starts every line with the
  server name, padded with `_`, so it can be searched with grep. `--blocks`
  gives the old layout. One server and runbooks keep their blocks.
- A server with no output shows `[ok]` in this layout.

## 2.3.0

- Searching works like grep everywhere: a name only has to contain the
  pattern, and patterns are regular expressions. `--ignore-case` is grep -i.
- `msh packages list|find|updates|compare PATTERN`.
- `msh hosts list|task list|runbook list|history list PATTERN`.
- Changed: patterns used to be shell-style and had to match the whole name.
  `xerox*` still works, but `lib*ssl` must now be written `lib.*ssl`.

## 2.2.1

- Every remote command takes destinations as words:
  `msh packages list tmux root@db01`, `msh uptime web01 web02`.
- `msh packages list PATTERN` and `msh packages check REQ...` take their
  values as words too.
- Fixed: flags after a word were silently ignored, so
  `msh packages list tmux --host x` lost its `--host`.

## 2.2.0

- Password login, after keys: a prompt on the terminal, or `MSH_PASSWORD` for
  scripts. Works with the password and keyboard-interactive methods.
- Several servers ask once per login user.
- Passwords are only sent to servers with a verified host key, and never
  with `--insecure-ignore-host-key`.
- The connect timeout pauses while you type.

## 2.1.1

- Changed: in the ssh-style form `--host` adds servers to the destination.
  The first word after the flags is always a destination:
  `msh --host web01,web02 root@db01 uptime` runs on three servers. To use
  only the listed servers, run `msh exec --host LIST --command CMD`.

## 2.1.0

- msh works like an ssh client: `msh user@host`, `msh host command`, with
  raw output and the remote exit status, and no inventory needed.
- ssh flags: `-l`, `-p`, `-i`, `-J`, `-t`, `-T`.
- `--host` takes `a,b`, `"a b"` and `@FILE`, and entries such as
  `user@host:port` that are not in the inventory, on every command.
- `msh --host LIST command` runs in parallel; without a command each server
  opens in a tmux pane.
- `~/.ssh/config` is used for servers that are not in the inventory.
- Changed: an unknown first word is now a destination, unless it looks like
  a typo of a command.

## 2.0.0

- Runbooks: multi-step workflows with dry run, retries, continue_on_error,
  timeouts, safety classes, arguments and resume after failure.
- Rolling execution: batches, canaries, failure limits, pauses, approvals
  and staggered starts.
- Facts: `msh facts`, a cache, `--expect` and `--group-by`.
- Expressions: a small language for `--where` and runbook `when:`.
- Packages: `msh packages list|check|find|compare|updates` for rpm and dpkg,
  with ported version comparison, requirements and baseline files.
- History: every run is recorded; `msh history list|show|diff|prune`.
- Shell completion for bash and zsh, a man page and a command reference,
  both generated from the program.
- Fixed: a command that exits 0 without reading all of its stdin is now a
  success, not an error.
- JSON output: `summary` has a new `skipped` count, and results of runbooks
  have `steps`.
- New status `skipped`.

## 1.0.0

First complete version of msh — My Parallel SSH, built in 150 layered
learning steps.

- Foundation: module layout, version and help, tests, check script, Makefile.
- CLI: subcommands, typed exit codes, typo suggestions, repeatable flags.
- Inventory: YAML/JSON, validation, defaults, selection by host, group, tag, all.
- Configuration: layered precedence (flag > environment > inventory defaults
  > config file > default), `config show`, `config validate`, `config init`.
- Execution: executor interface, local execution with process groups,
  output limits, timeouts.
- SSH: keys and agent, known_hosts verification, precise error kinds,
  in-process test server.
- Bastions: direct-tcpip tunnelling, per-bastion credentials and host keys,
  distinct errors, no fallback.
- Tasks: direct, shell and script modes, aliases, positional and named
  arguments, defaults, patterns, safety classes, injection-proof quoting.
- Runs: bounded worker pool, completion order or ordered output, fail-fast,
  per-server and global timeouts, CTRL+C cancellation, race-tested.
- Shortcuts, `logs` (with follow), local script streaming.
- Interactive sessions with PTY, raw mode, resize and signals; tmux panes.
- Output: text, prefix, JSON, CSV, JSONL, report files; structured logging.
- Release: version injection, cross-builds with checksums, documentation.
