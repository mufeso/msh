//go:build !unix

package interactive

// Resizes is not supported without SIGWINCH; the channel never fires.
func (t *stdTerminal) Resizes() (<-chan struct{}, func()) {
	ch := make(chan struct{})
	return ch, func() { close(ch) }
}
