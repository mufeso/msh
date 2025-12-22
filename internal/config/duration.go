package config

import (
	"encoding"
	"fmt"
	"time"
)

// Duration is a time.Duration that is written as text such as "10s" or
// "1m30s" in configuration files.
//
// encoding/json and yaml.v3 both use encoding.TextMarshaler and
// encoding.TextUnmarshaler when a type implements them, so these two
// methods make Duration work in both formats.
type Duration time.Duration

// Compile-time interface checks.
var (
	_ encoding.TextMarshaler   = Duration(0)
	_ encoding.TextUnmarshaler = (*Duration)(nil)
)

// MarshalText implements encoding.TextMarshaler.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(text []byte) error {
	v, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q (use for example 30s, 2m, 1h30m)", text)
	}
	*d = Duration(v)
	return nil
}

// String implements fmt.Stringer.
func (d Duration) String() string {
	return time.Duration(d).String()
}
