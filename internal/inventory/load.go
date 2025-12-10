package inventory

import (
	"fmt"
	"io"

	"example.com/msh/internal/decode"
)

// Parse reads an inventory in JSON format from r.
func Parse(r io.Reader) (*Inventory, error) {
	return ParseFormat(r, decode.JSON)
}

// ParseFormat reads an inventory in the given format from r. Unknown
// fields are rejected: a typo such as "adress" must be an error.
func ParseFormat(r io.Reader, format decode.Format) (*Inventory, error) {
	var inv Inventory
	if err := decode.Reader(r, format, &inv); err != nil {
		return nil, fmt.Errorf("parsing inventory: %w", err)
	}
	return &inv, nil
}

// LoadFile opens, parses, validates and completes the inventory file at
// path (.yaml, .yml or .json). fallbackUser is used for servers without a
// user.
func LoadFile(path, fallbackUser string) (*Inventory, error) {
	var inv Inventory
	if err := decode.File(path, &inv); err != nil {
		return nil, err
	}
	if err := inv.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	inv.ApplyDefaults(fallbackUser)
	return &inv, nil
}
