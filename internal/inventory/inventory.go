// Package inventory models the servers msh manages and how to select them.
//
// This is a domain package: it knows nothing about SSH, flags or output.
// Other packages depend on it; it depends only on the standard library.
package inventory

import (
	"net"
	"slices"
	"strconv"
)

// Server is one managed Linux host.
type Server struct {
	Name    string            // unique, used on the command line
	Address string            // DNS name or IP address
	User    string            // remote login user
	Port    int               // SSH port; 0 means "use the default"
	Groups  []string          // e.g. web, production
	Tags    map[string]string // e.g. environment=production
}

// HostPort returns "address:port", bracketing IPv6 addresses as required
// by net.Dial ("[2001:db8::1]:22").
//
// It uses a value receiver because it only reads the server.
func (s Server) HostPort() string {
	return net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
}

// InGroup reports whether the server belongs to group.
func (s Server) InGroup(group string) bool {
	return slices.Contains(s.Groups, group)
}

// HasTags reports whether the server has every key=value pair in want.
// An empty want matches every server.
func (s Server) HasTags(want map[string]string) bool {
	for k, v := range want {
		if got, ok := s.Tags[k]; !ok || got != v {
			return false
		}
	}
	return true
}

// Defaults are values applied to servers that do not set them.
type Defaults struct {
	User string
	Port int
}

// Inventory is the complete set of servers.
type Inventory struct {
	Defaults Defaults
	Servers  []Server
}

// Find returns the server with the given name.
// The second result follows the "comma ok" idiom used by maps.
func (inv *Inventory) Find(name string) (Server, bool) {
	for _, s := range inv.Servers {
		if s.Name == name {
			return s, true
		}
	}
	return Server{}, false
}

// Names returns all server names in inventory order.
func (inv *Inventory) Names() []string {
	names := make([]string, 0, len(inv.Servers))
	for _, s := range inv.Servers {
		names = append(names, s.Name)
	}
	return names
}
