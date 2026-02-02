// Package configfile describes the msh configuration file as a whole.
//
// The settings part belongs to package config and the tasks part to
// package task. Neither imports the other; this small package imports
// both, which keeps the dependency graph free of cycles:
//
//	configfile -> config
//	configfile -> task -> config
package configfile

import (
	"example.com/msh/internal/config"
	"example.com/msh/internal/decode"
	"example.com/msh/internal/task"
)

// File is the content of a configuration file.
type File struct {
	config.Layer `yaml:",inline"`
	Tasks        map[string]task.Task `json:"tasks,omitempty" yaml:"tasks,omitempty"`
}

// Load reads a configuration file (.yaml, .yml or .json).
func Load(path string) (File, error) {
	var f File
	if err := decode.File(path, &f); err != nil {
		return File{}, err
	}
	return f, nil
}
