# Changelog

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
