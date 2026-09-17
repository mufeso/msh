// Package tunnel forwards TCP connections. It knows nothing about SSH: the
// two ends are net.Listener and a dial function, so everything here can be
// tested with plain sockets.
package tunnel

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// loopback is the default bind address: only this machine can connect.
const loopback = "127.0.0.1"

// everywhere is what an empty or "*" bind address means, as in ssh.
const everywhere = "0.0.0.0"

var hostName = regexp.MustCompile(`^[A-Za-z0-9_:][A-Za-z0-9._:%-]*$`)

// Spec is one forward: listen on BindHost:BindPort, connect to Host:Port.
// Host and Port are empty for a dynamic forward.
type Spec struct {
	BindHost string
	BindPort int
	Host     string
	Port     int
}

// BindAddr is the address to listen on.
func (s Spec) BindAddr() string {
	return net.JoinHostPort(s.BindHost, strconv.Itoa(s.BindPort))
}

// Target is the address to connect to.
func (s Spec) Target() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

// String returns the spec in a form for messages.
func (s Spec) String() string {
	if s.Host == "" {
		return s.BindAddr()
	}
	return s.BindAddr() + ":" + s.Target()
}

// Public reports whether other machines can reach the listener.
func (s Spec) Public() bool {
	if s.BindHost == "localhost" {
		return false
	}
	ip := net.ParseIP(s.BindHost)
	return ip == nil || !ip.IsLoopback()
}

// ParseForward parses [bind:]port:host:hostport.
func ParseForward(text string) (Spec, error) {
	fields, err := split(text)
	if err != nil {
		return Spec{}, err
	}
	bind := loopback
	switch len(fields) {
	case 3:
	case 4:
		bind, fields = bindHost(fields[0]), fields[1:]
	default:
		return Spec{}, fmt.Errorf("invalid forward %q: use [bind:]port:host:hostport", text)
	}
	s := Spec{BindHost: bind, Host: fields[1]}
	if s.BindPort, err = port(fields[0], 0); err != nil {
		return Spec{}, fmt.Errorf("invalid forward %q: %w", text, err)
	}
	if s.Port, err = port(fields[2], 1); err != nil {
		return Spec{}, fmt.Errorf("invalid forward %q: %w", text, err)
	}
	if !hostName.MatchString(s.Host) || !hostName.MatchString(s.BindHost) {
		return Spec{}, fmt.Errorf("invalid forward %q: invalid host", text)
	}
	return s, nil
}

// ParseDynamic parses [bind:]port.
func ParseDynamic(text string) (Spec, error) {
	fields, err := split(text)
	if err != nil {
		return Spec{}, err
	}
	bind := loopback
	switch len(fields) {
	case 1:
	case 2:
		bind, fields = bindHost(fields[0]), fields[1:]
	default:
		return Spec{}, fmt.Errorf("invalid forward %q: use [bind:]port", text)
	}
	s := Spec{BindHost: bind}
	if s.BindPort, err = port(fields[0], 0); err != nil {
		return Spec{}, fmt.Errorf("invalid forward %q: %w", text, err)
	}
	if !hostName.MatchString(s.BindHost) {
		return Spec{}, fmt.Errorf("invalid forward %q: invalid host", text)
	}
	return s, nil
}

// ParseTarget parses host:port, as for ssh -W.
func ParseTarget(text string) (string, error) {
	host, p, err := net.SplitHostPort(text)
	if err != nil {
		return "", fmt.Errorf("invalid target %q: use host:port", text)
	}
	if _, err := port(p, 1); err != nil || !hostName.MatchString(host) {
		return "", fmt.Errorf("invalid target %q: use host:port", text)
	}
	return net.JoinHostPort(host, p), nil
}

func bindHost(s string) string {
	if s == "" || s == "*" {
		return everywhere
	}
	return s
}

func port(s string, lowest int) (int, error) {
	p, err := strconv.Atoi(s)
	if err != nil || p < lowest || p > 65535 {
		return 0, fmt.Errorf("invalid port %q", s)
	}
	return p, nil
}

// split cuts at colons, except inside [brackets], which it removes.
func split(text string) ([]string, error) {
	var fields []string
	rest := text
	for {
		if inner, ok := strings.CutPrefix(rest, "["); ok {
			field, after, found := strings.Cut(inner, "]")
			if !found {
				return nil, errors.New("missing ] in " + strconv.Quote(text))
			}
			fields = append(fields, field)
			if after == "" {
				return fields, nil
			}
			var colon bool
			if rest, colon = strings.CutPrefix(after, ":"); !colon {
				return nil, errors.New("unexpected text after ] in " + strconv.Quote(text))
			}
			continue
		}
		field, after, found := strings.Cut(rest, ":")
		fields = append(fields, field)
		if !found {
			return fields, nil
		}
		rest = after
	}
}
