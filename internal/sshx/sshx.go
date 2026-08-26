// Package sshx connects to servers over SSH using golang.org/x/crypto/ssh.
//
// It is an adapter: it translates msh's domain model (inventory servers,
// settings) into SSH client configuration, and it hides the concrete
// x/crypto types behind small interfaces so the rest of msh can be tested
// without a network.
package sshx

import (
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/inventory"
)

// clientVersion identifies msh in the SSH protocol banner. It contains no
// version number, so it reveals nothing useful to an attacker.
const clientVersion = "SSH-2.0-msh"

// Config holds connection settings shared by all servers of one run.
// It never contains private key material, only file paths.
type Config struct {
	IdentityFile          string        // explicit private key file ("" = agent and default keys)
	DefaultIdentityFiles  []string      // tried when IdentityFile is empty; missing files are skipped
	KnownHostsFile        string        // known_hosts used for host key verification
	InsecureIgnoreHostKey bool          // UNSAFE: skip host key verification
	ConnectTimeout        time.Duration // TCP connect + SSH handshake
	AgentSocket           string        // path of the ssh-agent socket (usually $SSH_AUTH_SOCK)
	Password              PasswordFunc  // nil = keys only
}

// Endpoint is one SSH destination.
type Endpoint struct {
	Name string // inventory name, used in messages
	Host string // DNS name or IP address
	Port int
	User string
}

// EndpointFor converts an inventory server (with defaults applied) into
// an Endpoint.
func EndpointFor(s inventory.Server) Endpoint {
	return Endpoint{Name: s.Name, Host: s.Address, Port: s.Port, User: s.User}
}

// Addr returns "host:port" in the form net.Dial expects.
func (e Endpoint) Addr() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

// String returns "user@host:port" for messages.
func (e Endpoint) String() string {
	return e.User + "@" + e.Addr()
}

// baseClientConfig returns the settings shared by every connection.
// Authentication and host key checking are added in later steps.
func baseClientConfig(cfg Config, ep Endpoint) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:          ep.User,
		ClientVersion: clientVersion,
		Timeout:       cfg.ConnectTimeout,
	}
}
