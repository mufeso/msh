//go:build !unix

package history

// withLock runs fn without locking on platforms without flock.
func (s Store) withLock(fn func() error) error { return fn() }
