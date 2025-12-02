package app

import (
	"errors"
	"io/fs"
	"os"
	"os/user"

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
	inv, err := inventory.LoadFile(opts.inventoryPath, currentUser())
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

// currentUser returns the local login name, used as the last-resort SSH
// user. os/user can fail in minimal containers, so $USER is a fallback.
func currentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return os.Getenv("USER")
}
