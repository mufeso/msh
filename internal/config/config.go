// Package config holds msh settings and resolves them from several layers.
//
// Precedence, from highest to lowest (completed over the next steps):
//
//	command-line flag > environment variable > configuration file > built-in default
package config

// Setting keys, as written in configuration files. They also name the
// entries in Settings.Sources.
const (
	KeyInventory      = "inventory"
	KeyUser           = "user"
	KeyPort           = "port"
	KeyIdentityFile   = "identity_file"
	KeyKnownHostsFile = "known_hosts_file"
	KeyOutput         = "output"
)

// Source names used in Settings.Sources.
const (
	SourceDefault = "default"
	SourceFile    = "config file"
	SourceFlag    = "flag"
)

// Settings are fully resolved values: every field has a usable value.
type Settings struct {
	Inventory      string
	User           string
	Port           int
	IdentityFile   string
	KnownHostsFile string
	Output         string

	// Sources maps each setting key to the layer that provided its value.
	Sources map[string]string
}

// Defaults returns the built-in defaults, the lowest layer.
func Defaults() Settings {
	s := Settings{
		Port:   22,
		Output: "text",
	}
	s.Sources = map[string]string{}
	for _, k := range Keys() {
		s.Sources[k] = SourceDefault
	}
	return s
}

// Keys returns every setting key in display order.
func Keys() []string {
	return []string{KeyInventory, KeyUser, KeyPort, KeyIdentityFile, KeyKnownHostsFile, KeyOutput}
}

// Layer is a partial configuration. A nil pointer means "not set here".
type Layer struct {
	Inventory      *string `json:"inventory,omitempty" yaml:"inventory,omitempty"`
	User           *string `json:"user,omitempty" yaml:"user,omitempty"`
	Port           *int    `json:"port,omitempty" yaml:"port,omitempty"`
	IdentityFile   *string `json:"identity_file,omitempty" yaml:"identity_file,omitempty"`
	KnownHostsFile *string `json:"known_hosts_file,omitempty" yaml:"known_hosts_file,omitempty"`
	Output         *string `json:"output,omitempty" yaml:"output,omitempty"`
}

// Source is a named layer.
type Source struct {
	Name  string
	Layer Layer
}

// Resolve starts from Defaults and applies each source in order, so later
// sources win. Pass sources from lowest to highest precedence.
func Resolve(sources ...Source) Settings {
	s := Defaults()
	for _, src := range sources {
		s.Apply(src.Name, src.Layer)
	}
	return s
}

// Apply overrides s with every value that l sets and records name as the
// source of those values.
func (s *Settings) Apply(name string, l Layer) {
	if s.Sources == nil {
		s.Sources = map[string]string{}
	}
	set(s, name, KeyInventory, &s.Inventory, l.Inventory)
	set(s, name, KeyUser, &s.User, l.User)
	set(s, name, KeyPort, &s.Port, l.Port)
	set(s, name, KeyIdentityFile, &s.IdentityFile, l.IdentityFile)
	set(s, name, KeyKnownHostsFile, &s.KnownHostsFile, l.KnownHostsFile)
	set(s, name, KeyOutput, &s.Output, l.Output)
}

// set copies *src into *dst when src is not nil and records the source.
func set[T any](s *Settings, source, key string, dst *T, src *T) {
	if src != nil {
		*dst = *src
		s.Sources[key] = source
	}
}
