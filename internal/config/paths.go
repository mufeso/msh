package config

import "path/filepath"

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
