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
