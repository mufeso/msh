package task

import (
	"errors"
	"fmt"
	"strings"
)

// checkShellTemplate inspects a shell command template.
//
// It understands POSIX single quotes, double quotes and backslash escapes
// well enough to know whether a position is quoted. It reports:
//
//   - placeholders inside quotes: msh already quotes values, and a value
//     quoted again inside '...' would end the outer quotes;
//   - unquoted redirection operators when allowRedirection is false.
//
// It fails closed: an unterminated quote is an error.
func checkShellTemplate(tmpl string, allowRedirection bool) error {
	const (
		unquoted = iota
		single
		double
	)
	state := unquoted
	runes := []rune(tmpl)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if strings.HasPrefix(string(runes[i:]), "{{") && state != unquoted {
			return fmt.Errorf("placeholder at position %d is inside quotes; remove the quotes, msh quotes argument values itself", i)
		}
		switch state {
		case single:
			if r == '\'' {
				state = unquoted
			}
		case double:
			switch r {
			case '\\':
				i++ // skip the escaped character
			case '"':
				state = unquoted
			}
		default:
			switch r {
			case '\\':
				i++
			case '\'':
				state = single
			case '"':
				state = double
			case '<', '>':
				if !allowRedirection {
					return fmt.Errorf("redirection %q at position %d is not allowed; set allow_redirection: true if it is intended", string(r), i)
				}
			}
		}
	}
	if state != unquoted {
		return errors.New("unterminated quote in command")
	}
	return nil
}
