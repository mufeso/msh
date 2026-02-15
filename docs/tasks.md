# Tasks

A task is a named, reusable remote command defined under `tasks:` in the
configuration file.

```yaml
tasks:
  message:
    description: "Follow the messages log and filter by pattern"
    aliases: [msg]
    interpreter: bash
    command: "tail -100f /var/log/messages | grep --line-buffered -- {{pattern}}"
    arguments:
      - name: pattern
        required: true
    interactive: true

  restart-service:
    description: "Restart a systemd service"
    argv: [sudo, systemctl, restart, "{{service}}"]
    arguments:
      - name: service
        required: true
        pattern: "[A-Za-z0-9@._-]+"
    safety: changes
    timeout: 2m

  security-summary:
    description: "Collect a security summary"
    interpreter: bash
    script: |
      set -e
      echo "Host: $(hostname)"
      echo "Kernel: $(uname -r)"
      systemctl --failed --no-legend
```

Run tasks with:

```bash
msh task list
msh task show message
msh task run message --host server01 --arg sshd
msh task run restart-service --group web --param service=nginx --yes
msh c message server01 sshd          # short form
```

## Modes

| Field | Mode | How it runs | How arguments enter |
|-------|------|-------------|---------------------|
| `argv:` | direct | the program runs with separate arguments, no shell | substituted inside each word; each word is quoted |
| `command:` | shell | `INTERPRETER -c 'COMMAND'` (sh or bash) | each value becomes one single-quoted shell word |
| `script:` | script | the script is streamed to `INTERPRETER` on stdin (`bash -s --`, `python3 -`) | positional parameters: `$1`, `sys.argv[1]` |

Prefer `argv:` when you do not need shell features. Use `command:` for
pipelines. Use `script:` for anything longer than one line. Scripts are
never written to the remote disk.

## Arguments

Placeholders are `{{name}}` or `{{N}}` (1-based position). Values come
from `--arg VALUE` (in order) and `--param NAME=VALUE`. Each argument may
be `required`, have a `default`, and have a `pattern` (a regular
expression the whole value must match).

## Safety rules

- Argument values are never interpreted as shell syntax: they are
  inserted as single-quoted words or passed as positional parameters.
- Placeholders inside quotes (`'{{x}}'`, `"{{x}}"`) are rejected, because
  msh already quotes the value.
- Redirections (`<`, `>`, `>>`, `<<`) in shell commands are rejected
  unless `allow_redirection: true` is set.
- Placeholders are only allowed in `command:` for sh and bash; other
  interpreters must use `script:` and read their arguments.
- `safety: changes` and `safety: dangerous` tasks require `--yes`.
- The test suite sends hostile values (`$(...)`, backticks, `;`, `|`,
  quotes, globs, newlines) through every mode to a real shell and checks
  they arrive unchanged with no side effects.

The remote login shell must be POSIX-compatible (sh, bash, dash, zsh,
ksh). Fish or csh login shells are not supported for tasks.
