# Interactive sessions

```bash
msh shell --host server01                       # like: ssh user@server01
msh shell --host server01 --command "top"        # like: ssh -t user@server01 top
msh shell --host server01 --command "podman exec -it asdf /bin/bash"
msh shell --host server01 --no-tty --command "cat /etc/hostname"   # like ssh -T
msh c message server01 sshd                      # an interactive task
```

The command always runs on the server, never locally. Sessions use the
same inventory, keys, agent, known_hosts verification, timeouts and
bastions as every other msh command.

## Terminal handling

When msh's standard input and output are a terminal:

1. msh requests a pseudo-terminal (PTY) on the server with your `$TERM`
   and your current window size.
2. Your local terminal is switched to raw mode, so every key (CTRL+C,
   CTRL+D, arrows, tab) goes to the remote program.
3. When you resize the window, msh sends the new size (SIGWINCH →
   SSH window-change).
4. When the session ends, your terminal mode is restored, even on errors.

Without a local terminal (pipes, scripts) no PTY is requested, and remote
stdout and stderr stay separate. `--no-tty` forces that behavior.

## Exit status

Like ssh, msh exits with the remote program's exit status. Connection
problems use msh's own codes (4, 5, ...; see exit-codes.md).

## Signals

| Event | Effect |
|-------|--------|
| CTRL+C | sent to the remote program (raw mode) |
| terminal window closed (SIGHUP) | remote side gets SIGHUP, session closed, exit 130 |
| `kill -TERM` on msh | same as SIGHUP |

## Several servers

An interactive session needs exactly one server. Use `--tmux` (see
tmux.md) to open one pane per server.

## Limitations

- Local standard input is copied by a goroutine that may stay blocked in
  a terminal read until the next keypress or until msh exits; it never
  delays the end of the session.
- Programs that ignore SIGHUP (or run under nohup/screen on the server)
  keep running after the session ends.
