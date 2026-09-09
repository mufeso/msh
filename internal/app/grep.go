package app

import (
	"flag"

	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/textmatch"
)

// grepFlags are the flags of commands that search by pattern.
type grepFlags struct {
	ignoreCase bool
}

// The short form -i is not possible: settingsFlags reserves it for keys.
func (g *grepFlags) register(fs *flag.FlagSet) {
	fs.BoolVar(&g.ignoreCase, "ignore-case", false, "ignore case in the pattern (like grep -i)")
}

// compile prints the error of a bad pattern.
func (g *grepFlags) compile(c *cli, pattern string) (*textmatch.Matcher, bool) {
	m, err := textmatch.New(pattern, g.ignoreCase)
	if err != nil {
		c.errorf("%v", err)
		return nil, false
	}
	return m, true
}

// parseList parses a list command: flags and at most one pattern word.
// fallback, if not nil, is a flag that also holds the pattern.
func (g *grepFlags) parseList(c *cli, name string, fs *flag.FlagSet, args []string, fallback *string) (*textmatch.Matcher, exitcode.Code, bool) {
	words, code, ok := parseInterspersed(fs, args)
	if !ok {
		return nil, code, false
	}
	pattern := ""
	switch {
	case len(words) > 1:
		c.errorf("%s: only one pattern is allowed", name)
		return nil, exitcode.Usage, false
	case len(words) == 1:
		pattern = words[0]
	case fallback != nil:
		pattern = *fallback
	}
	m, ok := g.compile(c, pattern)
	if !ok {
		return nil, exitcode.Usage, false
	}
	return m, exitcode.OK, true
}
