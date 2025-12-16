package app

import (
	"errors"
	"io/fs"

	"example.com/msh/internal/config"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/output"
)

// loadSettings resolves the settings for this run from all layers. On
// failure it prints the error and returns ok=false with the exit code.
func (c *cli) loadSettings(opts *commonOptions) (s config.Settings, code exitcode.Code, ok bool) {
	var sources []config.Source
	if opts.configPath != "" {
		c.verbosef(opts, "loading configuration %s", opts.configPath)
		layer, err := config.LoadFile(opts.configPath)
		if err != nil {
			c.errorf("%v", err)
			return s, fileErrorCode(err), false
		}
		sources = append(sources, config.Source{Name: config.SourceFile, Layer: layer})
	}
	sources = append(sources, config.Source{Name: config.SourceFlag, Layer: opts.settingsLayer()})

	s = config.Resolve(sources...)
	if err := s.Validate(); err != nil {
		c.errorf("%v", err)
		return s, validationExitCode(s, err), false
	}
	opts.format = output.Format(s.Output)
	return s, exitcode.OK, true
}

// validationExitCode returns Usage when a bad value came from a flag (the
// user typed it just now) and Config when it came from a file.
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
