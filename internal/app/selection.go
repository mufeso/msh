package app

import (
	"errors"
	"flag"
	"fmt"

	"example.com/msh/internal/cliflag"
	"example.com/msh/internal/expr"
	"example.com/msh/internal/inventory"
)

// selectionOptions describe which servers a command acts on.
type selectionOptions struct {
	hosts  cliflag.HostList
	groups cliflag.StringList
	tags   cliflag.KeyValues
	all    bool
	where  string // expression over facts; see registerWhere
}

func (s *selectionOptions) register(fs *flag.FlagSet) {
	fs.Var(&s.hosts, "host", "servers: `a,b`, \"a b\", user@host or @FILE (repeatable)")
	fs.Var(&s.groups, "group", "group `name` (repeatable, comma-separated)")
	fs.Var(&s.tags, "tag", "tag filter `key=value` (repeatable)")
	fs.BoolVar(&s.all, "all", false, "select all servers")
}

// registerWhere adds --where. It is separate from register because only
// commands that go through prepareRun can collect facts.
func (s *selectionOptions) registerWhere(fs *flag.FlagSet) {
	fs.StringVar(&s.where, "where", "", "select servers by facts: an `expression` such as 'os_family == \"redhat\"'")
}

// isEmpty reports whether no selector was given.
func (s *selectionOptions) isEmpty() bool {
	return !s.all && len(s.hosts) == 0 && len(s.groups) == 0 && len(s.tags) == 0 && s.where == ""
}

// onlyHosts reports whether the selection names servers directly, which
// works without an inventory file.
func (s *selectionOptions) onlyHosts() bool {
	return len(s.hosts) > 0 && !s.all && len(s.groups) == 0 && len(s.tags) == 0
}

// validate rejects contradictory selections.
func (s *selectionOptions) validate() error {
	if s.all && (len(s.hosts) > 0 || len(s.groups) > 0 || len(s.tags) > 0) {
		return errors.New("--all cannot be combined with --host, --group or --tag")
	}
	if s.where != "" {
		if _, err := expr.Parse(s.where); err != nil {
			return fmt.Errorf("--where: %w", err)
		}
	}
	return nil
}

// selector converts the flags into the inventory's domain type.
func (s *selectionOptions) selector() inventory.Selector {
	// --where alone filters all servers.
	onlyWhere := s.where != "" && !s.all && len(s.hosts) == 0 && len(s.groups) == 0 && len(s.tags) == 0
	return inventory.Selector{All: s.all || onlyWhere, Hosts: s.hosts, Groups: s.groups, Tags: s.tags}
}
