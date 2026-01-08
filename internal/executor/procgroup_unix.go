//go:build unix

package executor

import (
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"
)

// killGrace is how long a canceled process group gets to exit after
// SIGTERM before it receives SIGKILL.
const killGrace = 2 * time.Second

// setProcessGroup makes cmd the leader of a new process group and
// arranges for the whole group to be terminated on cancellation.
//
// Without a process group, killing "sh -c 'sleep 30; echo'" kills only
// the shell; sleep keeps running and keeps the output pipe open.
//
// It returns a stop function that must be called after cmd finishes to
// release the pending SIGKILL timer.
func setProcessGroup(cmd *exec.Cmd) (stop func()) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// The timer is created by Cancel (in an os/exec goroutine) and stopped
	// by our caller, so access goes through an atomic pointer.
	var timer atomic.Pointer[time.Timer]
	cmd.Cancel = func() error {
		pgid := -cmd.Process.Pid // a negative PID addresses the group
		err := syscall.Kill(pgid, syscall.SIGTERM)
		timer.Store(time.AfterFunc(killGrace, func() {
			_ = syscall.Kill(pgid, syscall.SIGKILL)
		}))
		return err
	}
	// If a grandchild still holds stdout/stderr open after the process
	// exits, give up waiting for those pipes after this delay.
	cmd.WaitDelay = killGrace + time.Second

	return func() {
		if t := timer.Load(); t != nil {
			t.Stop()
		}
	}
}
