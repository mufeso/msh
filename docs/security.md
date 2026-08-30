# Security model

msh runs commands with your privileges on many servers at once, so it is
designed to be safe by default.

## Host keys

- Host keys are always verified against `known_hosts_file`
  (default `~/.ssh/known_hosts`), for targets and bastions.
- Unknown hosts and changed keys are errors (exit 4); a changed key is
  reported as a possible man-in-the-middle attack.
- `--insecure-ignore-host-key` disables verification only when given
  explicitly on the command line, prints a prominent warning every time,
  and cannot be set in configuration files or environment variables.

## Authentication

- Keys come first: ssh-agent (`ssh-add`), then key files. Encrypted keys must
  be in the agent; msh does not ask for passphrases.
- If keys don't work, a password is used: `MSH_PASSWORD`, or a prompt on the
  terminal without echo. Without either, msh fails instead of hanging.
- Passwords are never stored, are not accepted as flags or in configuration
  files or the inventory, and are not passed to tmux panes on a command line.
- A password is sent only after the host key was verified, and never with
  `--insecure-ignore-host-key`.
- `MSH_PASSWORD` goes to every server of the run, including bastions, whose
  keys were refused. Any process of the same user may be able to read an
  environment variable. Prefer keys.
- One password is tried once per server, so a wrong one for a large fleet
  counts one failed login on each server. Mind account lockout rules.
- Key files readable by other users are refused (like OpenSSH).
- Private keys stay in memory only for the run; agent keys never leave
  the agent.

## What msh never prints or logs

Passwords, passphrases, private keys and agent material never appear in
output, logs, errors or reports. Logs (`--verbose`, `--log-format json`)
contain metadata only: server names, statuses, exit codes, durations. They
never contain command output, scripts or task arguments.

## Remote commands

- msh is read-only unless you ask for a command: shortcuts only run
  read-only commands; tasks classified `changes` or `dangerous` require
  `--yes`.
- Task arguments are never interpreted as shell syntax (see tasks.md);
  hostile values are tested against a real shell.
- Scripts are streamed over SSH stdin and never written to the remote
  disk. Report files are written atomically with mode 0600.
- Log paths are restricted to `allowed_log_paths`.

## Data stored on your machine

msh keeps data only under your home directory, readable only by you: run
history (`~/.local/state/msh/history`), runbook progress and cached facts
(`~/.cache/msh/facts`). The history includes command lines and command
output. See history.md for how to limit or disable it.

## Bastions

msh tunnels through bastions with `direct-tcpip`; nothing runs on the
bastion, the target's host key is verified end to end, and a failing
bastion never triggers a direct connection.

## tmux

msh never modifies tmux sessions it did not create; it checks its owner
marker before killing anything.

## Repository hygiene

The repository contains no real hostnames, addresses, users or keys.
Tests generate throwaway keys and run in-process SSH servers on
127.0.0.1.
