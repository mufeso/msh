package inventory

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Parse reads an inventory in JSON format from r.
//
// Unknown fields are rejected: a typo such as "adress" should be an error,
// not a silently ignored field.
func Parse(r io.Reader) (*Inventory, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var inv Inventory
	if err := dec.Decode(&inv); err != nil {
		return nil, fmt.Errorf("parsing inventory: %w", err)
	}
	return &inv, nil
}

// LoadFile opens, parses, validates and completes the inventory file at
// path. fallbackUser is used for servers without a user.
func LoadFile(path, fallbackUser string) (*Inventory, error) {
	f, err := os.Open(path)
	if err != nil {
		// os.Open already returns a *fs.PathError that names the file.
		return nil, err
	}
	// defer runs when LoadFile returns, on every path, so the file is
	// always closed. Close errors do not matter for a read-only file.
	defer f.Close()

	inv, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := inv.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	inv.ApplyDefaults(fallbackUser)
	return inv, nil
}
