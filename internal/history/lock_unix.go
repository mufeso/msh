//go:build unix

package history

import (
	"os"
	"path/filepath"
	"syscall"
)

// withLock runs fn while holding an exclusive lock on the history. flock
// is advisory: it only coordinates programs that also take it. The kernel
// releases it when the process ends, so a crash can't leave a stale lock.
func (s Store) withLock(fn func() error) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(s.Dir, ".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}
