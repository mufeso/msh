package app

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"strings"

	"example.com/msh/internal/config"
	"example.com/msh/internal/configfile"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
	"example.com/msh/internal/runbook"
	"example.com/msh/internal/task"
)

// setup is everything a command needs after its flags are parsed.
type setup struct {
	configFile string               // configuration file that was loaded ("" = none)
	settings   config.Settings      // fully resolved and validated
	inv        *inventory.Inventory // nil unless the command needs servers
	tasks      *task.Set            // tasks from the configuration file
	runbooks   *runbook.Set         // runbooks from the configuration file
}

// load resolves settings and, if needInventory is true, loads the
// inventory. On failure it prints the error and returns a nil setup and
// the exit code.
//
// Precedence of layers, lowest first:
//
//	built-in defaults < config file < inventory defaults < environment < flags
//
// The inventory path itself is a setting, so loading happens in two
// phases: resolve without the inventory to find it, then resolve again
// with the inventory's "defaults:" block included.
func (c *cli) load(opts *commonOptions, needInventory bool) (*setup, exitcode.Code) {
	st := &setup{}
	home, _ := c.lookupEnv("HOME")

	var fileLayer *config.Source
	var taskDefs map[string]task.Task
	var runbookDefs map[string]runbook.Runbook
	path, explicit := c.configPath(opts)
	if path != "" {
		file, err := configfile.Load(path)
		switch {
		case err == nil:
			c.verbosef(opts, "loaded configuration %s", path)
			st.configFile = path
			fileLayer = &config.Source{Name: config.SourceFile, Layer: file.Layer}
			taskDefs = file.Tasks
			runbookDefs = file.Runbooks
		case !explicit && errors.Is(err, fs.ErrNotExist):
			c.verbosef(opts, "no configuration file at %s", path)
		default:
			c.errorf("%v", err)
			return nil, fileErrorCode(err)
		}
	}
	tasks, err := task.NewSet(taskDefs)
	if err != nil {
		c.errorf("%s: %v", path, err)
		return nil, exitcode.Config
	}
	st.tasks = tasks
	// Runbooks reference tasks, so they are validated after them.
	runbooks, err := runbook.NewSet(runbookDefs, tasks)
	if err != nil {
		c.errorf("%s: %v", path, err)
		return nil, exitcode.Config
	}
	st.runbooks = runbooks

	envLayer, err := config.EnvLayer(c.lookupEnv)
	if err != nil {
		c.errorf("environment: %v", err)
		return nil, exitcode.Config
	}

	// resolve combines the layers in precedence order. inv may be nil.
	resolve := func(inv *config.Layer) config.Settings {
		var sources []config.Source
		if fileLayer != nil {
			sources = append(sources, *fileLayer)
		}
		if inv != nil {
			sources = append(sources, config.Source{Name: config.SourceInventory, Layer: *inv})
		}
		sources = append(sources,
			config.Source{Name: config.SourceEnv, Layer: envLayer},
			config.Source{Name: config.SourceFlag, Layer: opts.settingsLayer()},
		)
		s := config.Resolve(sources...)
		s.ExpandPaths(home)
		return s
	}

	st.settings = resolve(nil)
	if needInventory {
		inv, code := c.loadInventory(opts, st.settings)
		if inv == nil {
			return nil, code
		}
		st.inv = inv
		st.settings = resolve(&inv.Defaults.Layer)
	}

	if err := st.settings.Validate(); err != nil {
		c.errorf("%v", err)
		return nil, validationExitCode(st.settings, err)
	}
	if st.inv != nil {
		user := st.settings.User
		if user == "" {
			user = currentUser()
		}
		st.inv.ApplyDefaults(user, st.settings.Port)
		if code := c.resolveBastions(st); code != exitcode.OK {
			return nil, code
		}
	}
	opts.format = output.Format(st.settings.Output)
	opts.history = st.settings.History
	opts.historyLimit = st.settings.HistoryLimit
	return st, exitcode.OK
}

// loadInventory reads and validates the inventory file named by s.
func (c *cli) loadInventory(opts *commonOptions, s config.Settings) (*inventory.Inventory, exitcode.Code) {
	c.verbosef(opts, "loading inventory %s", s.Inventory)
	inv, err := inventory.LoadFile(s.Inventory)
	if err == nil {
		return inv, exitcode.OK
	}
	if errors.Is(err, fs.ErrNotExist) && s.Sources[config.KeyInventory] == config.SourceDefault && opts.inventoryOptional {
		opts.noInventoryFile = true
		return &inventory.Inventory{}, exitcode.OK
	}
	if errors.Is(err, fs.ErrNotExist) && s.Sources[config.KeyInventory] == config.SourceDefault {
		c.errorf("no inventory file at %s: use --inventory FILE, %s, or \"inventory\" in the configuration file",
			s.Inventory, config.EnvInventory)
		return nil, exitcode.Config
	}
	c.errorf("%v", err)
	return nil, fileErrorCode(err)
}

// configPath returns the configuration file to load and whether the user
// asked for it explicitly (an explicit file must exist).
func (c *cli) configPath(opts *commonOptions) (path string, explicit bool) {
	if opts.configPath != "" {
		return opts.configPath, true
	}
	if v, ok := c.lookupEnv(config.EnvConfig); ok && v != "" {
		return v, true
	}
	return config.DefaultConfigPath(c.lookupEnv), false
}

// validationExitCode returns Usage when a bad value came from a flag (the
// user typed it just now) and Config otherwise.
func validationExitCode(s config.Settings, err error) exitcode.Code {
	var verr *config.ValidationError
	if errors.As(err, &verr) {
		for _, p := range verr.Problems {
			if s.Sources[p.Key] == config.SourceFlag {
				return exitcode.Usage
			}
		}
	}
	return exitcode.Config
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

// currentUser returns the local login name, used as the last-resort SSH
// user. os/user can fail in minimal containers, so $USER is a fallback.
func currentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return os.Getenv("USER")
}

// resolveBastions decides which bastion each server uses. A bastion
// given on the command line or in MSH_BASTION overrides the servers' own
// settings; one from the inventory defaults or config file does not.
func (c *cli) resolveBastions(st *setup) exitcode.Code {
	var override, fallback string
	switch st.settings.Sources[config.KeyBastion] {
	case config.SourceFlag, config.SourceEnv:
		override = st.settings.Bastion
	default:
		fallback = st.settings.Bastion
	}
	if err := addBastionTarget(st.inv, override); err != nil {
		c.errorf("--bastion: %v", err)
		return exitcode.Usage
	}
	if err := st.inv.ResolveBastions(override, fallback); err != nil {
		c.errorf("%v", err)
		if st.settings.Sources[config.KeyBastion] == config.SourceFlag {
			return exitcode.Usage
		}
		return exitcode.Config
	}
	return exitcode.OK
}

// addBastionTarget registers an ad hoc bastion such as jump@host:2222.
// A plain word must be a name from the inventory, so a typo stays an error.
func addBastionTarget(inv *inventory.Inventory, value string) error {
	if _, known := inv.FindBastion(value); known || !strings.ContainsAny(value, "@.:") {
		return nil
	}
	t, err := inventory.ParseTarget(value)
	if err != nil {
		return err
	}
	inv.Bastions = append(inv.Bastions, inventory.Bastion{Name: value, Address: t.Host, User: t.User, Port: t.Port})
	return nil
}
