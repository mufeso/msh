// Package decode strictly decodes JSON and YAML documents into Go values.
package decode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format is a document format.
type Format int

// Supported formats.
const (
	JSON Format = iota // iota numbers constants 0, 1, 2, ...
	YAML
)

// FormatForPath picks the format from the file extension. Unknown
// extensions are treated as YAML, which is also a superset of JSON.
func FormatForPath(path string) Format {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return JSON
	default:
		return YAML
	}
}

// Reader decodes one document from r into v. Unknown fields are errors in
// both formats, so typos in configuration files are reported.
func Reader(r io.Reader, format Format, v any) error {
	switch format {
	case JSON:
		dec := json.NewDecoder(r)
		dec.DisallowUnknownFields()
		return dec.Decode(v)
	case YAML:
		dec := yaml.NewDecoder(r)
		dec.KnownFields(true)
		err := dec.Decode(v)
		if errors.Is(err, io.EOF) {
			return errors.New("empty document")
		}
		return err
	default:
		return fmt.Errorf("unknown format %d", format)
	}
}

// File opens path and decodes it into v using the format implied by its
// extension. Open errors are returned unchanged (*fs.PathError).
func File(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := Reader(f, FormatForPath(path), v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
