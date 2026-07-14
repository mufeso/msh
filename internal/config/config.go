// Package config holds msh settings and resolves them from several layers.
//
// Precedence, from highest to lowest:
//
//	command-line flag > environment variable > configuration file > built-in default
package config

import (
	"strconv"
	"strings"
	"time"
)

// Setting keys, as written in configuration files. They also name the
// entries in Settings.Sources.
const (
	KeyInventory       = "inventory"
	KeyUser            = "user"
	KeyPort            = "port"
	KeyIdentityFile    = "identity_file"
	KeyKnownHostsFile  = "known_hosts_file"
	KeyOutput          = "output"
	KeyConnectTimeout  = "connect_timeout"
	KeyCommandTimeout  = "command_timeout"
	KeyTimeout         = "timeout"
	KeyMode            = "mode"
	KeyWorkers         = "workers"
	KeyOrdered         = "ordered"
	KeyBastion         = "bastion"
	KeyAllowedLogPaths = "allowed_log_paths"
	KeyTmuxLayout      = "tmux_layout"
	KeyFactsTTL        = "facts_ttl"
	KeyHistory         = "history"
)

// Execution modes.
const (
	ModeSequential = "sequential"
	ModeParallel   = "parallel"
)

// Source names used in Settings.Sources.
const (
	SourceDefault   = "default"
	SourceFile      = "config file"
	SourceInventory = "inventory defaults"
	SourceFlag      = "flag"
)

// Settings are fully resolved values: every field has a usable value.
type Settings struct {
	Inventory      string `json:"inventory"`
	User           string `json:"user"`
	Port           int    `json:"port"`
	IdentityFile   string `json:"identity_file"`
	KnownHostsFile string `json:"known_hosts_file"`
	Output         string `json:"output"`

	ConnectTimeout time.Duration `json:"connect_timeout"` // TCP connect + SSH handshake, per server
	CommandTimeout time.Duration `json:"command_timeout"` // remote command, per server (0 = none)
	Timeout        time.Duration `json:"timeout"`         // whole run, all servers (0 = none)

	Mode    string `json:"mode"`    // "sequential" or "parallel"
	Workers int    `json:"workers"` // maximum concurrent servers in parallel mode
	Ordered bool   `json:"ordered"` // print results in selection order instead of completion order

	// Bastion is the default jump host ("" or "none" = direct).
	Bastion string `json:"bastion"`

	// AllowedLogPaths are the directories "msh logs" may read from.
	AllowedLogPaths []string `json:"allowed_log_paths"`

	// TmuxLayout is the default layout for "msh shell --tmux".
	TmuxLayout string `json:"tmux_layout"`

	// FactsTTL is how long collected facts stay valid (0 = no cache).
	FactsTTL time.Duration `json:"facts_ttl"`

	// History records runs under the state directory.
	History bool `json:"history"`

	// Sources maps each setting key to the layer that provided its value.
	Sources map[string]string `json:"-"`
}

// Defaults returns the built-in defaults, the lowest layer.
func Defaults() Settings {
	s := Settings{
		Inventory:       "~/.config/msh/inventory.yaml",
		Port:            22,
		KnownHostsFile:  "~/.ssh/known_hosts",
		Output:          "text",
		ConnectTimeout:  10 * time.Second,
		CommandTimeout:  60 * time.Second,
		Mode:            ModeParallel,
		AllowedLogPaths: []string{"/var/log"},
		TmuxLayout:      "tiled",
		FactsTTL:        time.Hour,
		History:         true,
		Workers:         5,
	}
	s.Sources = map[string]string{}
	for _, k := range Keys() {
		s.Sources[k] = SourceDefault
	}
	return s
}

// Keys returns every setting key in display order.
func Keys() []string {
	return []string{
		KeyInventory, KeyUser, KeyPort, KeyIdentityFile, KeyKnownHostsFile, KeyOutput,
		KeyConnectTimeout, KeyCommandTimeout, KeyTimeout,
		KeyMode, KeyWorkers, KeyOrdered, KeyBastion, KeyAllowedLogPaths, KeyTmuxLayout, KeyFactsTTL, KeyHistory,
	}
}

// Layer is a partial configuration. A nil pointer means "not set here".
type Layer struct {
	Inventory      *string   `json:"inventory,omitempty" yaml:"inventory,omitempty"`
	User           *string   `json:"user,omitempty" yaml:"user,omitempty"`
	Port           *int      `json:"port,omitempty" yaml:"port,omitempty"`
	IdentityFile   *string   `json:"identity_file,omitempty" yaml:"identity_file,omitempty"`
	KnownHostsFile *string   `json:"known_hosts_file,omitempty" yaml:"known_hosts_file,omitempty"`
	Output         *string   `json:"output,omitempty" yaml:"output,omitempty"`
	ConnectTimeout *Duration `json:"connect_timeout,omitempty" yaml:"connect_timeout,omitempty"`
	CommandTimeout *Duration `json:"command_timeout,omitempty" yaml:"command_timeout,omitempty"`
	Timeout        *Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Mode           *string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Workers        *int      `json:"workers,omitempty" yaml:"workers,omitempty"`
	Ordered        *bool     `json:"ordered,omitempty" yaml:"ordered,omitempty"`
	Bastion        *string   `json:"bastion,omitempty" yaml:"bastion,omitempty"`
	// A nil slice means "not set"; an empty list allows nothing.
	AllowedLogPaths []string  `json:"allowed_log_paths,omitempty" yaml:"allowed_log_paths,omitempty"`
	TmuxLayout      *string   `json:"tmux_layout,omitempty" yaml:"tmux_layout,omitempty"`
	FactsTTL        *Duration `json:"facts_ttl,omitempty" yaml:"facts_ttl,omitempty"`
	History         *bool     `json:"history,omitempty" yaml:"history,omitempty"`
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
	setDuration(s, name, KeyConnectTimeout, &s.ConnectTimeout, l.ConnectTimeout)
	setDuration(s, name, KeyCommandTimeout, &s.CommandTimeout, l.CommandTimeout)
	setDuration(s, name, KeyTimeout, &s.Timeout, l.Timeout)
	set(s, name, KeyMode, &s.Mode, l.Mode)
	set(s, name, KeyWorkers, &s.Workers, l.Workers)
	set(s, name, KeyOrdered, &s.Ordered, l.Ordered)
	set(s, name, KeyBastion, &s.Bastion, l.Bastion)
	if l.AllowedLogPaths != nil {
		paths := append([]string(nil), l.AllowedLogPaths...)
		set(s, name, KeyAllowedLogPaths, &s.AllowedLogPaths, &paths)
	}
	set(s, name, KeyTmuxLayout, &s.TmuxLayout, l.TmuxLayout)
	setDuration(s, name, KeyFactsTTL, &s.FactsTTL, l.FactsTTL)
	set(s, name, KeyHistory, &s.History, l.History)
}

// set copies *src into *dst when src is not nil and records the source.
func set[T any](s *Settings, source, key string, dst *T, src *T) {
	if src != nil {
		*dst = *src
		s.Sources[key] = source
	}
}

// setDuration is set for Duration values, which convert to time.Duration.
func setDuration(s *Settings, source, key string, dst *time.Duration, src *Duration) {
	if src != nil {
		v := time.Duration(*src)
		set(s, source, key, dst, &v)
	}
}

// Value returns the setting named key formatted for display.
func (s Settings) Value(key string) string {
	switch key {
	case KeyInventory:
		return s.Inventory
	case KeyUser:
		return s.User
	case KeyPort:
		return strconv.Itoa(s.Port)
	case KeyIdentityFile:
		return s.IdentityFile
	case KeyKnownHostsFile:
		return s.KnownHostsFile
	case KeyOutput:
		return s.Output
	case KeyConnectTimeout:
		return s.ConnectTimeout.String()
	case KeyCommandTimeout:
		return s.CommandTimeout.String()
	case KeyTimeout:
		return s.Timeout.String()
	case KeyMode:
		return s.Mode
	case KeyWorkers:
		return strconv.Itoa(s.Workers)
	case KeyOrdered:
		return strconv.FormatBool(s.Ordered)
	case KeyBastion:
		return s.Bastion
	case KeyAllowedLogPaths:
		return strings.Join(s.AllowedLogPaths, ":")
	case KeyTmuxLayout:
		return s.TmuxLayout
	case KeyFactsTTL:
		return s.FactsTTL.String()
	case KeyHistory:
		return strconv.FormatBool(s.History)
	default:
		return ""
	}
}
