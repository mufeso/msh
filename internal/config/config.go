// Package config holds msh settings and resolves them from several layers.
//
// Precedence, from highest to lowest (completed over the next steps):
//
//	command-line flag > environment variable > configuration file > built-in default
package config

// Settings are fully resolved values: every field has a usable value.
type Settings struct {
	Inventory      string
	User           string
	Port           int
	IdentityFile   string
	KnownHostsFile string
	Output         string
}

// Defaults returns the built-in defaults, the lowest layer.
func Defaults() Settings {
	return Settings{
		Port:   22,
		Output: "text",
	}
}

// Layer is a partial configuration, for example the content of a config
// file. A nil pointer means "this layer does not set the value", which is
// different from setting it to "" or 0.
type Layer struct {
	Inventory      *string `json:"inventory,omitempty"`
	User           *string `json:"user,omitempty"`
	Port           *int    `json:"port,omitempty"`
	IdentityFile   *string `json:"identity_file,omitempty"`
	KnownHostsFile *string `json:"known_hosts_file,omitempty"`
	Output         *string `json:"output,omitempty"`
}

// Apply overrides s with every value that l sets.
func (s *Settings) Apply(l Layer) {
	set(&s.Inventory, l.Inventory)
	set(&s.User, l.User)
	set(&s.Port, l.Port)
	set(&s.IdentityFile, l.IdentityFile)
	set(&s.KnownHostsFile, l.KnownHostsFile)
	set(&s.Output, l.Output)
}

// set copies *src into *dst when src is not nil.
//
// It is generic: T is a type parameter, so the same function works for
// string, int, bool and any other type without duplicating code.
func set[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}
