# Runbooks

A task is one action. A runbook is an operation: several actions in a
fixed order, run on each selected server.

```yaml
runbooks:
  restart-web:
    description: Restart a service and verify it
    timeout: 10m                 # total limit per server
    arguments:
      - name: service
        default: nginx
        pattern: "[A-Za-z0-9@._-]+"
    steps:
      - name: check disk
        command: "df -hP /"
        read_only: true
      - name: restart
        task: restart-service
        params: {service: "{{service}}"}
      - name: verify
        command: "systemctl is-active {{service}}"
        read_only: true
        retries: 5
        retry_delay: 3s
```

```bash
msh runbook list
msh runbook show restart-web
msh runbook run restart-web --group web --dry-run
msh runbook run restart-web --group web --yes
msh runbook run restart-web --group web --yes --resume
```

## Steps

Each step has a `name` and either `task` (with `params` and `args`) or
`command` (run with `sh -c`, or `interpreter: bash`).

| Field | Meaning |
|-------|---------|
| `retries`, `retry_delay` | run a failed step again |
| `timeout` | limit for this step; overrides the task's timeout and `command_timeout` |
| `continue_on_error` | a failure does not stop the runbook; it stays visible in the step results |
| `read_only` | declares that a command changes nothing |
| `allow_redirection` | permits `<` and `>` in a command |

On each server the steps run in order. The first failure stops that
server and the remaining steps are reported as skipped; other servers
continue. Servers run in parallel like any msh command (`--mode`,
`--workers`, `--ordered`, `--fail-fast`, `--timeout`). CTRL+C stops
everything and is never retried.

## Arguments

`arguments:` work like task arguments (`required`, `default`, `pattern`).
Pass them with `--arg VALUE` or `--param NAME=VALUE` and use them as
`{{name}}` in commands, `params` and `args`. Values are always quoted and
can't inject shell syntax.

## Safety

A runbook is as dangerous as its most dangerous step. A command step
counts as `changes` unless it says `read_only: true`. Anything that is not
read-only needs `--yes`. `--dry-run` prints the exact commands and connects
to nothing.

## Rolling execution

Without rollout settings, all selected servers run together, limited by
`--workers`. With them, msh works through the fleet in stages:

```yaml
runbooks:
  restart-web:
    canary: 1          # first, one server alone
    batch: 25%         # then a quarter of the servers at a time
    max_failures: 0    # stop after more than this many failed servers
    pause: 30s         # wait between batches
    stagger: 2s        # gap between server starts
    approve: true      # ask before each next batch
    steps: [...]
```

| Setting | Flag | Meaning |
|---------|------|---------|
| `batch` | `--batch` | servers per batch: a number or a percentage (rounded up) |
| `canary` | `--canary` | size of a smaller first batch |
| `max_failures` | `--max-failures` | a number or percentage; default 0 with `batch` or `canary`, otherwise no limit |
| `pause` | `--pause` | wait between batches |
| `stagger` | `--stagger` | gap between server starts |
| `approve` | `--approve`, `--auto-approve` | ask on the terminal before each next batch |

A batch must finish before the next starts. When the limit is exceeded,
or an approval is refused, the remaining servers are not touched: they are
reported with status `skipped` and the run exits non-zero. Then fix the
problem and continue with `--resume`. `--dry-run` shows the batches.

## Resume

msh records each server's finished steps in
`~/.local/state/msh/runbooks/NAME.json` (mode 0600). After a failure, fix
the problem and add `--resume`: finished steps are skipped on each server.
The state is tied to the plan, so a changed runbook or different arguments
can't resume by accident. A fully successful run removes the state.

Each step opens its own SSH connection. Output appears when a server has
finished all its steps.
