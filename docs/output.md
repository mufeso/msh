# Output formats

Choose with `--output FORMAT` (or `output:` in the configuration,
`MSH_OUTPUT`). Results are written to **stdout**; diagnostics, warnings
and `--verbose` messages always go to **stderr**, so machine-readable
output is never mixed with logs.

| Format | Best for | Streaming |
|--------|----------|-----------|
| `text` | people and grep | yes; see below |
| `json` | scripts that read everything at the end | no, one document |
| `jsonl` | pipelines (`jq`), long runs | yes, one line per server, then a summary line |
| `csv` | spreadsheets and reports | yes, one row per server |

Results appear in completion order; `--ordered` switches to selection
order.

## Text layout

With several servers every line starts with the server name, padded with `_`
to the longest name, so grep still shows where a line came from. A server
that did not succeed has a line starting with `ERROR:` instead, with its name
inside, in every text layout, so `grep ERROR:` finds them all:

```
$ msh disk --host web01,server01 | grep /boot
web01___: /dev/sda2   2.0G  201M  1.6G  11% /boot
server01: /dev/sda2   2.0G  201M  1.6G  11% /boot
```

The last line is the summary, with the name column filled with `#`:

```
web01___: ...
server01: ...
########: 2 servers: 2 ok in 160ms
```

The padding is `_` and not spaces so the name stays one column for `awk` and
`cut`. Remote stderr lines say `name [stderr]:`. A server with no output shows
`name: [ok]`. For example:

```
ERROR: [failed: web01: exit 2]
ERROR: [timeout: 192.0.2.7: connection timed out: context deadline exceeded]
ERROR: [error: db09: DNS lookup failed: no such host]
```

`--blocks` prints one block per server with its status and time instead.
That is also the layout for one server and for runbooks, because it shows
their steps. `--prefix` forces the line layout for them.

## Result fields

| JSON / JSONL | CSV column | Meaning |
|--------------|------------|---------|
| `server` | server | inventory name |
| `address` | address | address used |
| `status` | status | ok, failed, timeout, canceled, error |
| `exit_code` | exit_code | remote exit code, -1 if none |
| `stdout`, `stderr` | stdout, stderr | captured output (up to 1 MiB each) |
| `stdout_truncated`, `stderr_truncated` | — | output was cut |
| `error` | error | why the command could not run |
| `started` | started | RFC 3339 UTC |
| `duration_ms` | duration_ms | per-server duration |

## Documents

- `json`: `{"results": [...], "summary": {"total", "ok", "failed", "error", "timeout", "canceled", "problems", "duration_ms"}}`
- `jsonl`: lines `{"type":"result", ...}` and a final `{"type":"summary", ...}`
- `csv`: a header row, then one row per server

`hosts list`, `task list` and `config show` support all formats. Commands
that print one object (`version`, `task show`, `config validate`) support
text, json and jsonl.
