# Output formats

Choose with `--output FORMAT` (or `output:` in the configuration,
`MSH_OUTPUT`). Results are written to **stdout**; diagnostics, warnings
and `--verbose` messages always go to **stderr**, so machine-readable
output is never mixed with logs.

| Format | Best for | Streaming |
|--------|----------|-----------|
| `text` | people | yes, one block per server (`--prefix`: one line per output line) |
| `json` | scripts that read everything at the end | no, one document |
| `jsonl` | pipelines (`jq`), long runs | yes, one line per server, then a summary line |
| `csv` | spreadsheets and reports | yes, one row per server |

Results appear in completion order; `--ordered` switches to selection
order.

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
