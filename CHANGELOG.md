# Changelog

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
