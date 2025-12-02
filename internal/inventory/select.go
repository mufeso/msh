package inventory

import (
	"errors"
	"fmt"
)

// ErrUnknownHost is returned when a selected server is not in the
// inventory. Callers test for it with errors.Is, which also sees through
// wrapping.
var ErrUnknownHost = errors.New("unknown host")

// Selector describes which servers to act on.
type Selector struct {
	Hosts []string // server names
}

// Select returns the selected servers.
func (inv *Inventory) Select(sel Selector) ([]Server, error) {
	// Build a name -> server index once, so each lookup is a map access
	// instead of a scan over all servers.
	byName := make(map[string]Server, len(inv.Servers))
	for _, s := range inv.Servers {
		byName[s.Name] = s
	}

	var result []Server
	for _, name := range sel.Hosts {
		s, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownHost, name)
		}
		result = append(result, s)
	}
	return result, nil
}
