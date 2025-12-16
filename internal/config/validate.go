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

// Validate checks resolved settings.
func (s Settings) Validate() error {
	verr := &ValidationError{}
	if s.Port < 1 || s.Port > 65535 {
		verr.add(KeyPort, "%d is out of range 1-65535", s.Port)
	}
	switch s.Output {
	case "text", "json":
	default:
		verr.add(KeyOutput, "%q is not one of text, json", s.Output)
	}
	if len(verr.Problems) > 0 {
		return verr
	}
	return nil
}
