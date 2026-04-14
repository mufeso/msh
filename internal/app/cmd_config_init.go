package app

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

// templates holds the starter files. The //go:embed directive makes the
// compiler include them in the binary, so "msh config init" works without
// any files next to the executable.
//
//go:embed templates/config.yaml templates/inventory.yaml
var templates embed.FS

const configInitUsage = `Usage: msh config init [--dir DIR] [--force]

Writes a starter config.yaml and inventory.yaml (mode 0600) into DIR,
by default ~/.config/msh. Existing files are never overwritten unless
--force is given.
`

func runConfigInit(c *cli, args []string) exitcode.Code {
	flags := c.newFlagSet("config init", configInitUsage)
	dir := flags.String("dir", "", "target `directory` (default ~/.config/msh)")
	force := flags.Bool("force", false, "overwrite existing files")
	if code, ok := parseFlags(flags, args); !ok {
		return code
	}
	if *dir == "" {
		home, ok := c.lookupEnv("HOME")
		if !ok || home == "" {
			c.errorf("config init: HOME is not set; use --dir")
			return exitcode.Usage
		}
		*dir = filepath.Join(home, ".config", "msh")
	}
	if err := os.MkdirAll(*dir, 0o700); err != nil {
		c.errorf("config init: %v", err)
		return exitcode.IO
	}
	for _, name := range []string{"config.yaml", "inventory.yaml"} {
		dst := filepath.Join(*dir, name)
		if _, err := os.Stat(dst); err == nil && !*force {
			c.errorf("config init: %s already exists (use --force to overwrite)", dst)
			return exitcode.Usage
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			c.errorf("config init: %v", err)
			return exitcode.IO
		}
		data, err := templates.ReadFile("templates/" + name)
		if err != nil {
			c.errorf("config init: %v", err)
			return exitcode.Internal
		}
		err = output.WriteFileAtomic(dst, 0o600, func(w io.Writer) error {
			_, err := w.Write(data)
			return err
		})
		if err != nil {
			c.errorf("config init: %v", err)
			return exitcode.IO
		}
		fmt.Fprintf(c.stdout, "wrote %s\n", dst)
	}
	fmt.Fprintln(c.stdout, "Edit the inventory, then run: msh config validate")
	return exitcode.OK
}
