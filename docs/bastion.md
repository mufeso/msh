# Bastion hosts in msh

A bastion (jump host) is an SSH server that msh connects to first in order
to reach servers that are not directly reachable:

```
msh ──SSH──▶ bastion ──TCP (direct-tcpip)──▶ target:22
 └────────────── end-to-end SSH to the target ──────────────┘
```

msh opens one SSH connection to the bastion and asks it to forward a TCP
connection to the target (a `direct-tcpip` channel, the same mechanism as
`ssh -J`). The SSH session with the target runs *inside* that tunnel, so:

- nothing is executed on the bastion,
- the target's host key is verified by msh, end to end,
- the bastion never sees the target session's plaintext.

## Configuration

```yaml
defaults:
  bastion: production-bastion      # optional default for all servers

bastions:
  - name: production-bastion
    address: bastion.example.com
    user: jumpuser                  # default: the resolved "user" setting
    port: 22                        # default: the resolved "port" setting
    identity_file: ~/.ssh/jump_key  # optional; default: the normal keys/agent
    known_hosts_file: ~/.ssh/known_hosts_bastion  # optional

servers:
  - name: app01
    address: 10.10.20.15
    user: appadmin
    bastion: production-bastion
  - name: edge01
    address: 192.0.2.50
    bastion: none                   # always direct
```

## Which bastion is used

```
--bastion NAME or MSH_BASTION      (command line)
  > server "bastion:"
    > default ("defaults: bastion:" in the inventory, or "bastion:" in the config file)
      > none (direct)
```

`none` at any level means "connect directly". `msh hosts list` shows the
route in the VIA column.

## No silent fallback

If a server is configured to use a bastion and the bastion fails, msh
reports the error and does **not** try a direct connection. A direct
attempt could bypass network policy or reach a different machine.

## Errors

Every connection error says which hop failed:

| Failure | Example message |
|---------|-----------------|
| bastion connection | `bastion jump: TCP connection failed: ...` |
| bastion authentication | `bastion jump: authentication failed: ...` |
| bastion host key | `bastion jump: host key verification failed: ...` |
| target connection | `app01 (via bastion jump): TCP connection failed: ...` |
| target authentication | `app01 (via bastion jump): authentication failed: ...` |
| target host key | `app01 (via bastion jump): host key verification failed: ...` |

All of them exit with code 4 (connection failure).

## Limitations

- One hop only. Chains of bastions are deliberately not supported to keep
  the design easy to reason about; use a single well-placed bastion.
- `connect_timeout` bounds the whole setup (bastion and target).
- The bastion must allow TCP forwarding (`AllowTcpForwarding yes` in
  sshd_config) to the target address.
