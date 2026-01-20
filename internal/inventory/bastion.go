package inventory

import (
	"fmt"
)

// NoBastion is the bastion name that forces a direct connection.
const NoBastion = "none"

// validateBastions checks the bastion list and every reference to it.
func (inv *Inventory) validateBastions(verr *ValidationError) {
	seen := map[string]bool{}
	for i, b := range inv.Bastions {
		where := fmt.Sprintf("bastions[%d]", i)
		if b.Name != "" {
			where = fmt.Sprintf("bastion %q", b.Name)
		}
		switch {
		case b.Name == "":
			verr.add("%s: name is required", where)
		case b.Name == NoBastion:
			verr.add("%s: %q is reserved", where, NoBastion)
		case !namePattern.MatchString(b.Name):
			verr.add("%s: name must match %s", where, namePattern)
		case seen[b.Name]:
			verr.add("%s: duplicate name", where)
		}
		seen[b.Name] = true
		if b.Address == "" {
			verr.add("%s: address is required", where)
		}
		if !validPort(b.Port) {
			verr.add("%s: port %d out of range 1-65535", where, b.Port)
		}
	}
	for _, s := range inv.Servers {
		if s.Bastion != "" && s.Bastion != NoBastion && !seen[s.Bastion] {
			verr.add("server %q: unknown bastion %q", s.Name, s.Bastion)
		}
	}
	if d := inv.Defaults.Bastion; d != nil && *d != "" && *d != NoBastion && !seen[*d] {
		verr.add("defaults: unknown bastion %q", *d)
	}
}

// ResolveBastions sets each server's Bastion field to the bastion msh
// will actually use, following this precedence:
//
//  1. override, from the command line (--bastion) or MSH_BASTION
//  2. the server's own "bastion:" field
//  3. fallback, the default from inventory defaults or the config file
//  4. none (direct connection)
//
// "none" at any level means "connect directly" and stops the search. The
// result is "" for a direct connection. Unknown names are errors: msh
// never silently falls back to a direct connection.
func (inv *Inventory) ResolveBastions(override, fallback string) error {
	for _, name := range []string{override, fallback} {
		if name != "" && name != NoBastion {
			if _, ok := inv.FindBastion(name); !ok {
				return fmt.Errorf("unknown bastion %q", name)
			}
		}
	}
	for i := range inv.Servers {
		inv.Servers[i].Bastion = pickBastion(override, inv.Servers[i].Bastion, fallback)
	}
	return nil
}

// pickBastion applies the precedence for one server.
func pickBastion(override, server, fallback string) string {
	for _, candidate := range []string{override, server, fallback} {
		switch candidate {
		case "":
			continue
		case NoBastion:
			return ""
		default:
			return candidate
		}
	}
	return ""
}
