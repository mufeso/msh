package app

import (
	"errors"
	"io/fs"

	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
)

// loadSettings resolves the settings for this run. On failure it prints
// the error and returns ok=false with the exit code to use.
func (c *cli) loadSettings(opts *commonOptions) (s config.Settings, code exitcode.Code, ok bool) {
	s = config.Defaults()
	if opts.configPath != "" {
		c.verbosef(opts, "loading configuration %s", opts.configPath)
		layer, err := config.LoadFile(opts.configPath)
		if err != nil {
			c.errorf("%v", err)
			return s, fileErrorCode(err), false
		}
		s.Apply(layer)
	}
	// The --inventory flag overrides the configuration file.
	if opts.inventoryPath != "" {
		s.Inventory = opts.inventoryPath
	}
	return s, exitcode.OK, true
}

// fileErrorCode maps a load error to an exit code: a *fs.PathError means
// the file could not be read (IO); anything else is invalid content.
func fileErrorCode(err error) exitcode.Code {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return exitcode.IO
	}
	return exitcode.Config
}
