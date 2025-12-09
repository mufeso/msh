package app

import (
	"os"
	"os/user"

	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
)

// loadInventory loads the inventory named by the settings. On failure it
// prints the error and returns a nil inventory and the exit code to use.
func (c *cli) loadInventory(opts *commonOptions, s config.Settings) (*inventory.Inventory, exitcode.Code) {
	if s.Inventory == "" {
		c.errorf("no inventory file: use --inventory FILE or set \"inventory\" in the configuration")
		return nil, exitcode.Config
	}
	fallbackUser := s.User
	if fallbackUser == "" {
		fallbackUser = currentUser()
	}
	c.verbosef(opts, "loading inventory %s", s.Inventory)
	inv, err := inventory.LoadFile(s.Inventory, fallbackUser)
	if err != nil {
		c.errorf("%v", err)
		return nil, fileErrorCode(err)
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
