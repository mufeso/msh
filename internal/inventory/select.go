package inventory

import (
	"errors"
	"fmt"
)

// ErrUnknownHost is returned when a selected server is not in the
// inventory.
var ErrUnknownHost = errors.New("unknown host")

// Selector describes which servers to act on.
type Selector struct {
	Hosts []string // server names, in the order the user gave them
}

// Select returns the selected servers. Each server appears at most once,
// in the order it was first selected.
func (inv *Inventory) Select(sel Selector) ([]Server, error) {
	byName := make(map[string]Server, len(inv.Servers))
	for _, s := range inv.Servers {
		byName[s.Name] = s
	}

	// A map with empty-struct values is Go's idiomatic set type.
	seen := make(map[string]struct{})
	var result []Server
	for _, name := range sel.Hosts {
		s, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownHost, name)
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, s)
	}
	return result, nil
}
