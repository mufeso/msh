package config

import (
	"fmt"
	"io"

	"example.com/msh/internal/decode"
)

// Parse reads a configuration layer in the given format.
func Parse(r io.Reader, format decode.Format) (Layer, error) {
	var l Layer
	if err := decode.Reader(r, format, &l); err != nil {
		return Layer{}, fmt.Errorf("parsing configuration: %w", err)
	}
	return l, nil
}

// LoadFile reads the configuration file at path (.yaml, .yml or .json).
func LoadFile(path string) (Layer, error) {
	var l Layer
	if err := decode.File(path, &l); err != nil {
		return Layer{}, err
	}
	return l, nil
}
