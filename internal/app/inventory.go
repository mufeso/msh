package app

import (
	"errors"
	"io/fs"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
)

// loadInventory loads the inventory named by the options. On failure it
// prints the error and returns a nil inventory and the exit code to use.
func (c *cli) loadInventory(opts *commonOptions) (*inventory.Inventory, exitcode.Code) {
	if opts.inventoryPath == "" {
		c.errorf("no inventory file: use --inventory FILE")
		return nil, exitcode.Config
	}
	c.verbosef(opts, "loading inventory %s", opts.inventoryPath)
	inv, err := inventory.LoadFile(opts.inventoryPath)
	if err != nil {
		c.errorf("%v", err)
		// A *fs.PathError means the file itself could not be read; anything
		// else means its content is invalid.
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil, exitcode.IO
		}
		return nil, exitcode.Config
	}
	return inv, exitcode.OK
}
