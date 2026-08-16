// Package cliflag provides custom flag types for the msh command line.
//
// The flag package accepts any type that implements flag.Value:
//
//	type Value interface {
//		String() string
//		Set(string) error
//	}
//
// Go interfaces are satisfied implicitly: a type only needs the methods.
package cliflag

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

// StringList collects values from repeated flags and comma-separated lists:
//
//	--host web01 --host web02,web03  =>  [web01 web02 web03]
type StringList []string

// Compile-time checks that the types implement flag.Value.
var (
	_ flag.Value = (*StringList)(nil)
	_ flag.Value = (*KeyValues)(nil)
)

// String returns the values joined by commas.
func (l *StringList) String() string {
	if l == nil {
		return ""
	}
	return strings.Join(*l, ",")
}

// Set is called by the flag package once per occurrence of the flag. It
// has a pointer receiver because it must modify the caller's slice.
func (l *StringList) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("empty item in %q", value)
		}
		*l = append(*l, part)
	}
	return nil
}

// KeyValues collects key=value pairs from repeated flags:
//
//	--tag env=prod --tag role=web  =>  map[env:prod role:web]
type KeyValues map[string]string

// String returns the pairs sorted by key, so output is deterministic.
func (kv *KeyValues) String() string {
	if kv == nil || *kv == nil {
		return ""
	}
	keys := make([]string, 0, len(*kv))
	for k := range *kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+(*kv)[k])
	}
	return strings.Join(parts, ",")
}

// Set parses one key=value pair. A nil map is created on first use.
func (kv *KeyValues) Set(value string) error {
	key, val, ok := strings.Cut(value, "=")
	key = strings.TrimSpace(key)
	if !ok || key == "" {
		return fmt.Errorf("%q is not in key=value form", value)
	}
	if *kv == nil {
		*kv = make(KeyValues)
	}
	(*kv)[key] = strings.TrimSpace(val)
	return nil
}

// HostList collects server names from commas, spaces, repeated flags and
// files:
//
//	--host a,b --host "c d" --host @servers.txt
type HostList []string

var _ flag.Value = (*HostList)(nil)

// String returns the hosts joined by commas.
func (l *HostList) String() string {
	if l == nil {
		return ""
	}
	return strings.Join(*l, ",")
}

// Set adds hosts. A value starting with "@" names a file; user@host has
// the "@" in the middle, so the two can't be confused.
func (l *HostList) Set(value string) error {
	if path, ok := strings.CutPrefix(value, "@"); ok {
		return l.ReadFile(path)
	}
	hosts := splitHosts(value)
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts in %q", value)
	}
	*l = append(*l, hosts...)
	return nil
}

// ReadFile adds the hosts of a file. Names are separated by lines, spaces
// or commas; everything after # on a line is a comment.
func (l *HostList) ReadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var hosts []string
	for _, line := range strings.Split(string(data), "\n") {
		line, _, _ = strings.Cut(line, "#")
		hosts = append(hosts, splitHosts(line)...)
	}
	if len(hosts) == 0 {
		return fmt.Errorf("%s: no hosts", path)
	}
	*l = append(*l, hosts...)
	return nil
}

// splitHosts splits on commas and any whitespace.
func splitHosts(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
}
