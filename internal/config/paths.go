package config

import (
	"path/filepath"
	"strings"
)

// DefaultConfigPath returns the configuration file used when neither
// --config nor MSH_CONFIG is given: $XDG_CONFIG_HOME/msh/config.yaml, or
// $HOME/.config/msh/config.yaml. It returns "" if neither variable is set.
func DefaultConfigPath(lookup LookupFunc) string {
	if dir, ok := lookup("XDG_CONFIG_HOME"); ok && dir != "" {
		return filepath.Join(dir, "msh", "config.yaml")
	}
	if home, ok := lookup("HOME"); ok && home != "" {
		return filepath.Join(home, ".config", "msh", "config.yaml")
	}
	return ""
}

// CacheDir returns the directory for data that can be rebuilt:
// $XDG_CACHE_HOME/msh, or $HOME/.cache/msh. It returns "" if neither
// variable is set.
func CacheDir(lookup LookupFunc) string {
	if dir, ok := lookup("XDG_CACHE_HOME"); ok && dir != "" {
		return filepath.Join(dir, "msh")
	}
	if home, ok := lookup("HOME"); ok && home != "" {
		return filepath.Join(home, ".cache", "msh")
	}
	return ""
}

// StateDir returns the directory for data msh keeps between runs:
// $XDG_STATE_HOME/msh, or $HOME/.local/state/msh. It returns "" if neither
// variable is set.
func StateDir(lookup LookupFunc) string {
	if dir, ok := lookup("XDG_STATE_HOME"); ok && dir != "" {
		return filepath.Join(dir, "msh")
	}
	if home, ok := lookup("HOME"); ok && home != "" {
		return filepath.Join(home, ".local", "state", "msh")
	}
	return ""
}

// ExpandHome replaces a leading "~" or "~/" in path with home. Other
// paths, including "~user/...", are returned unchanged.
func ExpandHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(home, rest)
	}
	return path
}

// ExpandPaths expands "~" in every path-valued setting.
func (s *Settings) ExpandPaths(home string) {
	s.Inventory = ExpandHome(s.Inventory, home)
	s.IdentityFile = ExpandHome(s.IdentityFile, home)
	s.KnownHostsFile = ExpandHome(s.KnownHostsFile, home)
}
