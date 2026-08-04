# Run history

msh saves a record of each run on the local machine, so you can look up
what happened and compare two runs.

```bash
msh history list
msh history list --failed --since 24h --server web01 --grep nginx
msh history show last
msh history show 20261105T1015
msh history diff 20261105T1015 last
msh history prune --keep 50
```

`REF` is `last`, a full ID or a unique beginning of an ID.

## What is recorded

Runs of `exec`, `task run`, `c`, `script run`, `logs`, shortcuts and
`runbook run`. Interactive sessions, `facts` and `packages` are not
recorded. `logs --follow` records only the status, not the streamed output.

Each record has the command line, the local user, the time, the summary
and, per server, status, exit code, duration, and up to 64 KiB of stdout and
stderr. For runbooks it also has every step.

## diff

Shows per server what changed: status, exit code and the first differing
line of stdout, and servers that appear in only one run. Timing is ignored.
It exits 1 if the runs differ. A useful habit: run the same read-only check
before and after a change.

## Storage and privacy

Records live in `~/.local/state/msh/history` (or `$XDG_STATE_HOME/msh`), one
JSON file per run: directory mode 0700, files 0600.

**Records can contain sensitive data**: the command line as you typed it
(including `--param` values) and the output of commands. Scripts sent with
`script run` are not stored, only their output. Secrets don't belong on a
command line; for a run where they can't be avoided, use `--no-history`.

| Setting | Environment | Default | Meaning |
|---------|-------------|---------|---------|
| `history` | `MSH_HISTORY` | `true` | record runs |
| `history_limit` | `MSH_HISTORY_LIMIT` | `200` | runs to keep (`0` = all) |

Failing to write a record prints a warning and never changes the result of
the run.

## File format

Records have a schema number. Older records are upgraded when they are
read; records from a newer msh are refused with a message.
