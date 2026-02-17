# msh exit codes

| Code | Name | Meaning |
|-----:|------|---------|
| 0 | OK | every selected server succeeded |
| 1 | Failure | at least one remote command exited non-zero |
| 2 | Usage | invalid command-line input (unknown flag, bad selection, missing --yes, bad task argument) |
| 3 | Config | invalid configuration, inventory or task definition |
| 4 | Connection | DNS, TCP, SSH handshake, authentication, host key or bastion failure |
| 5 | Timeout | a connection, command or the global timeout expired |
| 6 | Transfer | a script or binary could not be transferred |
| 7 | Tmux | tmux is missing or failed |
| 8 | IO | a local file could not be read or written |
| 70 | Internal | unexpected internal error (please report it) |
| 130 | Canceled | interrupted with CTRL+C (SIGINT) or SIGTERM |

When several servers end differently, the most significant outcome wins:

```
canceled > timeout > error (connection, transfer, internal) > failed > ok
```

For example, if one server was unreachable (4) and another returned a
non-zero exit (1), msh exits 4.
