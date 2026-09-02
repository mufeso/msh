package app

import (
	"flag"

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
