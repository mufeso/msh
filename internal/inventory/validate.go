package inventory

import (
	"fmt"
	"regexp"
	"strings"
)

// namePattern restricts server, group and tag names to characters that are
// safe on a command line and in file names. regexp.MustCompile panics on a
// bad pattern, which is fine for a constant pattern checked at start-up.
var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidationError lists every problem found in an inventory.
type ValidationError struct {
	Problems []string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return "invalid inventory:\n  - " + strings.Join(e.Problems, "\n  - ")
}

// add records one problem.
func (e *ValidationError) add(format string, a ...any) {
	e.Problems = append(e.Problems, fmt.Sprintf(format, a...))
}

// Validate checks the inventory and returns a *ValidationError describing
// all problems, or nil.
func (inv *Inventory) Validate() error {
	verr := &ValidationError{}

	if len(inv.Servers) == 0 {
		verr.add("no servers defined")
	}
	if inv.Defaults.Inventory != nil {
		verr.add("defaults: \"inventory\" cannot be set inside an inventory")
	}
	if p := inv.Defaults.Port; p != nil && (*p < 1 || *p > 65535) {
		verr.add("defaults: port %d out of range 1-65535", *p)
	}

	seen := make(map[string]int) // name -> index of first occurrence
	for i, s := range inv.Servers {
		where := fmt.Sprintf("servers[%d]", i)
		if s.Name != "" {
			where = fmt.Sprintf("server %q", s.Name)
		}

		switch {
		case s.Name == "":
			verr.add("%s: name is required", where)
		case !namePattern.MatchString(s.Name):
			verr.add("%s: name must match %s", where, namePattern)
		}
		if first, dup := seen[s.Name]; dup && s.Name != "" {
			verr.add("%s: duplicate name (also servers[%d])", where, first)
		} else {
			seen[s.Name] = i
		}

		if s.Address == "" {
			verr.add("%s: address is required", where)
		}
		if !validPort(s.Port) {
			verr.add("%s: port %d out of range 1-65535", where, s.Port)
		}
		for _, g := range s.Groups {
			if !namePattern.MatchString(g) {
				verr.add("%s: invalid group name %q", where, g)
			}
		}
		for k := range s.Tags {
			if !namePattern.MatchString(k) {
				verr.add("%s: invalid tag key %q", where, k)
			}
		}
	}

	if len(verr.Problems) > 0 {
		return verr
	}
	// Return a literal nil, not a nil *ValidationError: an interface that
	// holds a typed nil pointer is not equal to nil.
	return nil
}

// validPort accepts 0 (meaning "default") and 1-65535.
func validPort(p int) bool {
	return p >= 0 && p <= 65535
}
