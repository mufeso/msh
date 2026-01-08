//go:build !unix

package executor

import (
	"os/exec"
	"time"
)

// setProcessGroup is a no-op on platforms without POSIX process groups.
// os/exec still kills the direct child on cancellation.
func setProcessGroup(cmd *exec.Cmd) (stop func()) {
	cmd.WaitDelay = 3 * time.Second
	return func() {}
}
