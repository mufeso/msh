package app

import (
	"errors"
	"io/fs"

	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

// loadSettings resolves the settings for this run from all layers:
//
//	built-in defaults < config file < environment < flags
//
// On failure it prints the error and returns ok=false with the exit code.
func (c *cli) loadSettings(opts *commonOptions) (s config.Settings, code exitcode.Code, ok bool) {
	var sources []config.Source

	path, explicit := c.configPath(opts)
	if path != "" {
		layer, err := config.LoadFile(path)
		switch {
		case err == nil:
			c.verbosef(opts, "loaded configuration %s", path)
			sources = append(sources, config.Source{Name: config.SourceFile, Layer: layer})
		case !explicit && errors.Is(err, fs.ErrNotExist):
			// The default configuration file is optional.
			c.verbosef(opts, "no configuration file at %s", path)
		default:
			c.errorf("%v", err)
			return s, fileErrorCode(err), false
		}
	}

	envLayer, err := config.EnvLayer(c.lookupEnv)
	if err != nil {
		c.errorf("environment: %v", err)
		return s, exitcode.Config, false
	}
	sources = append(sources,
		config.Source{Name: config.SourceEnv, Layer: envLayer},
		config.Source{Name: config.SourceFlag, Layer: opts.settingsLayer()},
	)

	s = config.Resolve(sources...)
	if err := s.Validate(); err != nil {
		c.errorf("%v", err)
		return s, validationExitCode(s, err), false
	}
	opts.format = output.Format(s.Output)
	return s, exitcode.OK, true
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
