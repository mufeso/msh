# Running local scripts remotely

```bash
msh script run --host server01 --file ./check.sh
msh script run --group web --file ./health-check.py --interpreter python3
msh script run --host server01 --interpreter bash --file ./check.sh --arg nginx
```

## How it works

1. msh reads the local file once (regular file, at most 16 MiB).
2. For each server it opens an SSH session and starts the interpreter in
   "read the program from standard input" mode:

   | Interpreter | Remote command |
   |-------------|----------------|
   | sh | `sh -s -- ARGS...` |
   | bash | `bash -s -- ARGS...` |
   | python3 | `python3 - ARGS...` |
   | perl | `perl - ARGS...` |

3. The script is written to the session's standard input, followed by
   end-of-file. Output and the exit status come back as for any command.

Nothing is copied to the remote disk and no temporary files are created,
locally or remotely. The remote command line contains only the
interpreter and the quoted arguments, never the script text.

Without `--interpreter`, the shebang line decides (`#!/bin/bash` → bash,
`#!/usr/bin/env python3` → python3), otherwise sh.

## Arguments

`--arg VALUE` (repeatable) becomes `$1`, `$2`, ... in shell scripts and
`sys.argv[1:]` in Python. Values are quoted for the remote shell and
never inserted into the script body.

## Standard input

Because standard input carries the script, the script itself cannot read
interactive input. Commands inside the script that read stdin see the
rest of the script for sh/bash; redirect them from `/dev/null` if needed.

## Cancellation and cleanup

CTRL+C, `--command-timeout` and `--timeout` stop the script: msh sends
SIGTERM over SSH, closes the session and the connection. Processes the
script started in the background (`&`, `nohup`, `setsid`) may survive if
they ignore the hang-up; msh cannot reach them after the session is gone.
