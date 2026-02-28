# Concurrency in msh

## The run pipeline

```
                 ┌─────────── worker 1 ──┐
dispatcher ─jobs─┼─────────── worker 2 ──┼─out─▶ collector ─▶ emit ─▶ output sink
                 └─────────── worker N ──┘
```

- The **dispatcher** sends server indexes on the unbuffered `jobs` channel.
- **N workers** (`--workers`, default 5; `--mode sequential` means 1)
  receive jobs, run them through the `executor.Executor` interface and
  send results on `out`.
- The **collector** is the caller's goroutine. It stores each result at
  its index (so the report keeps selection order) and calls `emit`.
  Output sinks therefore never run concurrently and need no locks.
- Channels are unbuffered: if the terminal is slow, workers wait
  (backpressure) instead of piling up output in memory.

Goroutine count is bounded: N workers + 1 dispatcher + 1 closer, plus one
helper goroutine per running SSH session (waiting on `Session.Wait`).

## Ordering

- Default: **completion order** — a result is printed when its server
  finishes.
- `--ordered`: a reorder buffer keeps early results until every earlier
  server is done, then flushes them. Output still streams.

## Contexts

```
root (CTRL+C / SIGTERM)
 └── run (global --timeout)
      ├── scheduling (stopped by fail-fast or the run ending)
      └── per server
           ├── connect (connect_timeout, both hops of a bastion)
           └── command (command_timeout or a task's timeout)
```

- CTRL+C cancels the root. Workers stop picking up servers; running SSH
  commands receive SIGTERM, then the session and connection are closed;
  local commands' process groups receive SIGTERM, then SIGKILL after 2s.
- Every server is always reported: servers that never started get a
  "not started" result.
- A second CTRL+C exits immediately (code 130).

## Timeouts

| Setting | Scope | Result |
|---------|-------|--------|
| `connect_timeout` | TCP + SSH handshake (+ bastion), per server | `timeout`, "connection timed out" |
| `command_timeout` | remote command, per server | `timeout`, "remote command timed out" |
| `timeout` | whole run | `timeout`, "global timeout expired" (+ notice on stderr) |

## Cleanup guarantees

- Every SSH session and connection opened for a server is closed before
  that server's result is emitted (tests assert the server side sees no
  open sessions afterwards).
- The goroutine waiting on `Session.Wait` always ends: after
  cancellation the session is closed, which makes `Wait` return.
- `Run` returns only after all workers have exited (leak tests compare
  goroutine counts before and after).

## Limitations

Remote processes that ignore SIGTERM, detach from the session (`nohup`,
`setsid`, daemons) or run under a server that does not support SSH
signals may keep running after the session is closed. msh cannot stop
what it can no longer reach.
