package inventory

import (
	"errors"
	"fmt"
)

// Selection errors. Callers test for them with errors.Is.
var (
	ErrUnknownHost  = errors.New("unknown host")
	ErrUnknownGroup = errors.New("unknown group")
	ErrNoMatch      = errors.New("no servers match the selection")
)

// Selector describes which servers to act on.
//
// Semantics:
//   - All selects every server (the other fields must be empty).
//   - Otherwise the result is the servers named in Hosts (in that order)
//     followed by members of any group in Groups (in inventory order).
//   - Tags then filters the result: every key=value pair must match. If
//     only Tags is given, it filters all servers.
//   - Each server appears once. An empty result is ErrNoMatch.
type Selector struct {
	All    bool
	Hosts  []string
	Groups []string
	Tags   map[string]string
}

// IsEmpty reports whether the selector selects nothing explicitly.
func (sel Selector) IsEmpty() bool {
	return !sel.All && len(sel.Hosts) == 0 && len(sel.Groups) == 0 && len(sel.Tags) == 0
}

// Select returns the selected servers.
func (inv *Inventory) Select(sel Selector) ([]Server, error) {
	if sel.All {
		return inv.nonEmpty(append([]Server(nil), inv.Servers...))
	}

	byName := make(map[string]Server, len(inv.Servers))
	knownGroups := make(map[string]struct{})
	for _, s := range inv.Servers {
		byName[s.Name] = s
		for _, g := range s.Groups {
			knownGroups[g] = struct{}{}
		}
	}

	seen := make(map[string]struct{})
	var result []Server
	add := func(s Server) {
		if _, dup := seen[s.Name]; !dup {
			seen[s.Name] = struct{}{}
			result = append(result, s)
		}
	}

	for _, name := range sel.Hosts {
		s, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownHost, name)
		}
		add(s)
	}
	for _, g := range sel.Groups {
		if _, ok := knownGroups[g]; !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownGroup, g)
		}
		for _, s := range inv.Servers {
			if s.InGroup(g) {
				add(s)
			}
		}
	}

	if len(sel.Tags) > 0 {
		base := result
		if len(sel.Hosts) == 0 && len(sel.Groups) == 0 {
			base = inv.Servers
		}
		var filtered []Server
		for _, s := range base {
			if s.HasTags(sel.Tags) {
				filtered = append(filtered, s)
			}
		}
		result = filtered
	}
	return inv.nonEmpty(result)
}

// nonEmpty turns an empty selection into ErrNoMatch.
func (inv *Inventory) nonEmpty(servers []Server) ([]Server, error) {
	if len(servers) == 0 {
		return nil, ErrNoMatch
	}
	return servers, nil
}
