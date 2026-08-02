package config

import (
	"fmt"
	"strings"
)

// Problem is one invalid setting.
type Problem struct {
	Key     string // setting name as written in config files, e.g. "port"
	Message string
}

// ValidationError lists all invalid settings.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Problems))
	for _, p := range e.Problems {
		parts = append(parts, p.Key+": "+p.Message)
	}
	return "invalid settings: " + strings.Join(parts, "; ")
}

func (e *ValidationError) add(key, format string, a ...any) {
	e.Problems = append(e.Problems, Problem{Key: key, Message: fmt.Sprintf(format, a...)})
}

// MaxWorkers bounds concurrency: each worker holds an SSH connection, and
// too many at once can exhaust file descriptors or overload a bastion.
const MaxWorkers = 512

// Validate checks resolved settings.
func (s Settings) Validate() error {
	verr := &ValidationError{}
	if s.Port < 1 || s.Port > 65535 {
		verr.add(KeyPort, "%d is out of range 1-65535", s.Port)
	}
	switch s.Output {
	case "text", "json", "csv", "jsonl":
	default:
		verr.add(KeyOutput, "%q is not one of text, json, csv, jsonl", s.Output)
	}
	if s.ConnectTimeout <= 0 {
		verr.add(KeyConnectTimeout, "must be greater than 0")
	}
	if s.CommandTimeout < 0 {
		verr.add(KeyCommandTimeout, "must not be negative (0 means no limit)")
	}
	if s.HistoryLimit < 0 {
		verr.add(KeyHistoryLimit, "must not be negative (0 keeps all runs)")
	}
	if s.FactsTTL < 0 {
		verr.add(KeyFactsTTL, "must not be negative (0 disables the cache)")
	}
	if s.Timeout < 0 {
		verr.add(KeyTimeout, "must not be negative (0 means no limit)")
	}
	switch s.Mode {
	case ModeSequential, ModeParallel:
	default:
		verr.add(KeyMode, "%q is not one of %s, %s", s.Mode, ModeSequential, ModeParallel)
	}
	if s.Workers < 1 || s.Workers > MaxWorkers {
		verr.add(KeyWorkers, "%d is out of range 1-%d", s.Workers, MaxWorkers)
	}
	if len(verr.Problems) > 0 {
		return verr
	}
	return nil
}
