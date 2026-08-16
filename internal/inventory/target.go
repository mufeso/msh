package inventory

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Target is an SSH destination as a person writes it.
type Target struct {
	User string // "" = not given
	Host string
	Port int // 0 = not given
}

var (
	targetUser = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)
	targetHost = regexp.MustCompile(`^[A-Za-z0-9_:][A-Za-z0-9._:%-]*$`)
)

// ParseTarget parses host, user@host, user@host:port, [ipv6]:port or
// ssh://user@host:port. A name starting with "-" is rejected, so a
// destination can never turn into an option of something else.
func ParseTarget(s string) (Target, error) {
	var t Target
	rest := strings.TrimSuffix(strings.TrimPrefix(s, "ssh://"), "/")
	if i := strings.LastIndexByte(rest, '@'); i >= 0 {
		t.User, rest = rest[:i], rest[i+1:]
		if !targetUser.MatchString(t.User) {
			return Target{}, fmt.Errorf("invalid user in %q", s)
		}
	}

	port := ""
	switch {
	case strings.HasPrefix(rest, "["): // [2001:db8::1]:22
		host, after, ok := strings.Cut(rest[1:], "]")
		if !ok {
			return Target{}, fmt.Errorf("missing ] in %q", s)
		}
		rest = host
		if after != "" {
			var hasPort bool
			if port, hasPort = strings.CutPrefix(after, ":"); !hasPort {
				return Target{}, fmt.Errorf("unexpected text after ] in %q", s)
			}
		}
	case strings.Count(rest, ":") == 1: // host:22; more colons are bare IPv6
		rest, port, _ = strings.Cut(rest, ":")
	}
	t.Host = rest
	if !targetHost.MatchString(t.Host) {
		return Target{}, fmt.Errorf("invalid host in %q", s)
	}
	if port != "" {
		p, err := strconv.Atoi(port)
		if err != nil || p < 1 || p > 65535 {
			return Target{}, fmt.Errorf("invalid port in %q", s)
		}
		t.Port = p
	}
	return t, nil
}

// TargetDefaults complete ad hoc servers.
type TargetDefaults struct {
	User         string
	Port         int
	Bastion      string
	ForceBastion bool // the bastion came from the command line
}

// AddTargets makes every entry of hosts selectable and returns the names
// to select them by.
//
// An entry that names an inventory server, or its address, keeps that
// server's settings, and a user or port in the entry override them. Any
// other entry becomes an ad hoc server named as it was typed. resolve, if
// not nil, can complete a new server (for example from ~/.ssh/config).
func (inv *Inventory) AddTargets(hosts []string, d TargetDefaults, resolve func(*Server) error) ([]string, error) {
	names := make([]string, 0, len(hosts))
	known := len(inv.Servers) // ad hoc servers never merge with each other
	for _, entry := range hosts {
		if _, ok := inv.Find(entry); ok {
			names = append(names, entry)
			continue
		}
		t, err := ParseTarget(entry)
		if err != nil {
			return nil, err
		}
		if i := inv.indexOf(t, known); i >= 0 {
			if t.User != "" {
				inv.Servers[i].User = t.User
			}
			if t.Port != 0 {
				inv.Servers[i].Port = t.Port
			}
			names = append(names, inv.Servers[i].Name)
			continue
		}
		s := Server{Name: entry, Address: t.Host, User: t.User, Port: t.Port}
		if resolve != nil {
			if err := resolve(&s); err != nil {
				return nil, fmt.Errorf("%s: %w", entry, err)
			}
		}
		if s.User == "" {
			s.User = d.User
		}
		if s.Port == 0 {
			s.Port = d.Port
		}
		if d.ForceBastion || s.Bastion == "" {
			s.Bastion = d.Bastion
		}
		inv.Servers = append(inv.Servers, s)
		names = append(names, entry)
	}
	return names, nil
}

// indexOf finds a target among the first n servers, by name or by address.
// One address with another port is another server: several can share a
// host behind port forwarding.
func (inv *Inventory) indexOf(t Target, n int) int {
	for i, s := range inv.Servers[:n] {
		if s.Name == t.Host {
			return i
		}
		if s.Address == t.Host && (t.Port == 0 || s.Port == 0 || t.Port == s.Port) {
			return i
		}
	}
	return -1
}
