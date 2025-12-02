package app

import (
	"errors"
	"flag"

	"example.com/msh/internal/cliflag"
	"example.com/msh/internal/inventory"
)

// selectionOptions describe which servers a command acts on.
type selectionOptions struct {
	hosts  cliflag.StringList
	groups cliflag.StringList
	tags   cliflag.KeyValues
	all    bool
}

func (s *selectionOptions) register(fs *flag.FlagSet) {
	fs.Var(&s.hosts, "host", "server `name` (repeatable, comma-separated)")
	fs.Var(&s.groups, "group", "group `name` (repeatable, comma-separated)")
	fs.Var(&s.tags, "tag", "tag filter `key=value` (repeatable)")
	fs.BoolVar(&s.all, "all", false, "select all servers")
}

// isEmpty reports whether no selector was given.
func (s *selectionOptions) isEmpty() bool {
	return !s.all && len(s.hosts) == 0 && len(s.groups) == 0 && len(s.tags) == 0
}

// validate rejects contradictory selections.
func (s *selectionOptions) validate() error {
	if s.all && (len(s.hosts) > 0 || len(s.groups) > 0 || len(s.tags) > 0) {
		return errors.New("--all cannot be combined with --host, --group or --tag")
	}
	return nil
}

// selector converts the flags into the inventory's domain type.
func (s *selectionOptions) selector() inventory.Selector {
	return inventory.Selector{Hosts: s.hosts}
}
