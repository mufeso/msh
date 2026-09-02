// Package textmatch is msh's one rule for searching text: it works like
// grep. A pattern is a regular expression that can match anywhere in the
// text, so a plain word means "contains".
package textmatch

import (
	"fmt"
	"regexp"
	"strings"
)

// Matcher is a compiled pattern. Compile it once, before a loop.
type Matcher struct {
	re *regexp.Regexp
}

// New compiles pattern; an empty pattern matches everything. Leading "*"
// are dropped: they mean nothing in a search that matches anywhere, and
// '*ssl*' is a common habit from shell patterns.
func New(pattern string, ignoreCase bool) (*Matcher, error) {
	expr := strings.TrimLeft(pattern, "*")
	if ignoreCase {
		expr = "(?i)" + expr
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}
	return &Matcher{re: re}, nil
}

// Match reports whether text contains the pattern.
func (m *Matcher) Match(text string) bool { return m.re.MatchString(text) }
