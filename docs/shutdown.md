# Shutdown and cancellation

msh creates one root context when it starts. SIGINT (CTRL+C), SIGTERM
and SIGHUP cancel it; everything msh does derives from that context.

| Situation | First signal | Second signal |
|-----------|--------------|---------------|
| `exec`, `task run`, `script run`, shortcuts | stop scheduling servers; running remote commands get SIGTERM, then their sessions and connections are closed; every server is reported; exit 130 | immediate exit (130) |
| `logs --follow` | stop following on every server; exit 130 | immediate exit |
| `shell` (raw mode) | CTRL+C is a keystroke for the remote program; SIGTERM/SIGHUP hang up the session, restore the terminal; exit 130 | immediate exit |
| `shell --tmux` while panes are being created | stop creating panes; kill the session msh created (after checking its owner marker) | immediate exit |
| `exec --local` | the command's process group gets SIGTERM, then SIGKILL after 2 s | immediate exit |

## What is cleaned up

- SSH sessions, channels and connections (direct and bastion).
- Goroutines waiting on remote commands.
- Timers and child contexts.
- Local process groups.
- Temporary report files (only complete reports are renamed into place).
- tmux sessions created by msh during setup.

## Limits

msh cannot reach a server after the connection is gone. Remote processes
that ignore SIGTERM/SIGHUP, run under `nohup`/`setsid`, or are started as
services keep running. Scripts are never written to the remote disk, so a
lost connection never leaves script files behind.

## Exit codes

Run `msh help exit-codes` or see exit-codes.md.
