// Package inventory models the servers msh manages and how to select them.
//
// This is a domain package: it knows nothing about SSH, flags or output.
// Other packages depend on it; it depends only on the standard library.
package inventory

import (
	"net"

	"example.com/msh/internal/config"
	"slices"
	"strconv"
)

// Server is one managed Linux host.
//
// The struct tags name the fields in inventory files. "omitempty" leaves
// empty values out when the server is written back as JSON.
type Server struct {
	Name    string            `json:"name" yaml:"name"`                           // unique, used on the command line
	Address string            `json:"address" yaml:"address"`                     // DNS name or IP address
	User    string            `json:"user,omitempty" yaml:"user,omitempty"`       // remote login user
	Port    int               `json:"port,omitempty" yaml:"port,omitempty"`       // SSH port; 0 means "use the default"
	Groups  []string          `json:"groups,omitempty" yaml:"groups,omitempty"`   // e.g. web, production
	Tags    map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`       // e.g. environment=production
	Bastion string            `json:"bastion,omitempty" yaml:"bastion,omitempty"` // name of the jump host to connect through
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

// Defaults is the inventory's "defaults:" block. It is a configuration
// layer: it can set the same values as a configuration file (user, port,
// timeouts, workers, ...), and it is applied with higher precedence than
// the configuration file but lower than environment variables and flags.
//
// config.Layer is embedded. encoding/json promotes the fields of an
// embedded struct automatically; yaml.v3 needs the ",inline" tag.
type Defaults struct {
	config.Layer `yaml:",inline"`
}

// Inventory is the complete set of servers.
type Inventory struct {
	Defaults Defaults  `json:"defaults" yaml:"defaults"`
	Bastions []Bastion `json:"bastions,omitempty" yaml:"bastions,omitempty"`
	Servers  []Server  `json:"servers" yaml:"servers"`
}

// Bastion is an intermediate SSH host ("jump host"). msh connects to the
// bastion first and then opens a forwarded TCP connection (direct-tcpip)
// from the bastion to the target server; no command runs on the bastion.
type Bastion struct {
	Name           string `json:"name" yaml:"name"`
	Address        string `json:"address" yaml:"address"`
	User           string `json:"user,omitempty" yaml:"user,omitempty"`
	Port           int    `json:"port,omitempty" yaml:"port,omitempty"`
	IdentityFile   string `json:"identity_file,omitempty" yaml:"identity_file,omitempty"`
	KnownHostsFile string `json:"known_hosts_file,omitempty" yaml:"known_hosts_file,omitempty"`
}

// FindBastion returns the bastion with the given name.
func (inv *Inventory) FindBastion(name string) (Bastion, bool) {
	for _, b := range inv.Bastions {
		if b.Name == name {
			return b, true
		}
	}
	return Bastion{}, false
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
