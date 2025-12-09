package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Parse reads a configuration layer in JSON format.
func Parse(r io.Reader) (Layer, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var l Layer
	if err := dec.Decode(&l); err != nil {
		return Layer{}, fmt.Errorf("parsing configuration: %w", err)
	}
	return l, nil
}

// LoadFile reads the configuration file at path.
func LoadFile(path string) (Layer, error) {
	f, err := os.Open(path)
	if err != nil {
		return Layer{}, err
	}
	defer f.Close()

	l, err := Parse(f)
	if err != nil {
		return Layer{}, fmt.Errorf("%s: %w", path, err)
	}
	return l, nil
}
