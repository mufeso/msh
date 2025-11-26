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
	"sort"
	"strings"
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
