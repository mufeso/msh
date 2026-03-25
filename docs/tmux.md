# Several interactive sessions with tmux

```bash
msh shell --group web --tmux
msh shell --all --tmux --tmux-session fleet --tmux-layout tiled
msh task run message --group web --arg sshd --tmux     # interactive task in panes
msh shell --group web --tmux --tmux-detached           # create, print the name, do not attach
```

msh opens one tmux pane per selected server. Each pane runs this same
msh binary as `msh shell --host SERVER ...` with the run's resolved
settings passed explicitly (absolute inventory/config/known_hosts/key
paths, user, port, connect timeout, bastion route), so every pane has its
own SSH session to exactly one server. Pane borders show the server name.

## Requirements

- tmux in `PATH`. msh never installs software; without tmux it exits
  with code 7 and explains how to install it.
- An interactive terminal (unless `--tmux-detached`).
- At least two selected servers (for one, use `msh shell` directly).

## Options

| Flag | Setting | Default |
|------|---------|---------|
| `--tmux-session NAME` | — | random `msh-xxxxxxxx` |
| `--tmux-layout NAME` | `tmux_layout`, `MSH_TMUX_LAYOUT` | `tiled` |
| `--tmux-detached` | — | attach |

Layouts: tiled, even-horizontal, even-vertical, main-horizontal,
main-vertical.

Inside an existing tmux client, msh switches to the new session
(`switch-client`); otherwise it attaches (`attach-session`).

## Safety

- All tmux commands are run with argument lists (no shell). Pane commands
  are quoted word by word because tmux hands them to a shell.
- Session names are restricted to letters, digits, `-` and `_`, and all
  targets use tmux's exact-match form (`-t =NAME`).
- msh refuses to use a session name that already exists.
- Every session msh creates carries an `@msh-owner` marker unique to the
  run. msh only kills a session after checking that marker.

## Cleanup

- If creating panes fails, or CTRL+C arrives during setup, msh kills the
  session it created (after checking the marker) and stops launching
  panes.
- A pane whose msh session fails stays open until you press Enter, so
  you can read the error.
- When all panes have exited, tmux ends the session by itself.
- If you detach (`CTRL+B d`), the session keeps running on purpose; use
  `tmux attach -t NAME` to return or `tmux kill-session -t NAME` to end it.
- Unrelated tmux sessions are never modified.
