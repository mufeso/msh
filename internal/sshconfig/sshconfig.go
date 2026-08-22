// Package sshconfig reads the part of OpenSSH's client configuration that
// msh can use. Everything else in the file is ignored, and so are Match
// blocks, because their conditions can run commands.
package sshconfig

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const maxIncludeDepth = 8

// Host holds the options for one destination.
type Host struct {
	HostName      string
	User          string
	Port          int
	IdentityFiles []string
	ProxyJump     string
}

type option struct{ key, value string }

type block struct {
	patterns []string
	options  []option
}

// Config is a parsed configuration file.
type Config struct {
	home   string
	blocks []block
}

// Load reads a file and what it includes. home expands ~ and is where
// relative Include paths start, in home/.ssh.
func Load(file, home string) (*Config, error) {
	c := &Config{home: home}
	if err := c.loadFile(file, []string{"*"}, 0); err != nil {
		return nil, err
	}
	return c, nil
}

// Parse reads a configuration without following Include.
func Parse(r io.Reader, home string) (*Config, error) {
	c := &Config{home: home}
	if err := c.parse(r, "config", []string{"*"}, -1); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) loadFile(file string, patterns []string, depth int) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.parse(f, file, patterns, depth)
}

// parse adds the blocks of r. patterns apply until the first Host line, so
// a file included inside a Host block stays inside it. A negative depth
// ignores Include.
func (c *Config) parse(r io.Reader, name string, patterns []string, depth int) error {
	cur := block{patterns: patterns}
	skipping := false
	scanner := bufio.NewScanner(r)
	for n := 1; scanner.Scan(); n++ {
		key, args := splitLine(scanner.Text())
		if key == "" {
			continue
		}
		if len(args) == 0 {
			return fmt.Errorf("%s:%d: %s has no value", name, n, key)
		}
		switch key {
		case "host":
			c.blocks = append(c.blocks, cur)
			cur = block{patterns: args}
			skipping = false
		case "match":
			c.blocks = append(c.blocks, cur)
			cur = block{}
			skipping = true
		case "include":
			if skipping || depth < 0 {
				continue
			}
			c.blocks = append(c.blocks, cur)
			cur = block{patterns: cur.patterns}
			if err := c.include(args, cur.patterns, depth); err != nil {
				return fmt.Errorf("%s:%d: %w", name, n, err)
			}
		case "port":
			if p, err := strconv.Atoi(args[0]); err != nil || p < 1 || p > 65535 {
				return fmt.Errorf("%s:%d: invalid port %q", name, n, args[0])
			}
			cur.options = append(cur.options, option{key, args[0]})
		case "hostname", "user", "identityfile", "proxyjump":
			cur.options = append(cur.options, option{key, args[0]})
		}
	}
	c.blocks = append(c.blocks, cur)
	return scanner.Err()
}

func (c *Config) include(globs, patterns []string, depth int) error {
	if depth >= maxIncludeDepth {
		return errors.New("Include is nested too deeply")
	}
	for _, glob := range globs {
		glob = c.expandHome(glob)
		if !filepath.IsAbs(glob) {
			glob = filepath.Join(c.home, ".ssh", glob)
		}
		files, err := filepath.Glob(glob)
		if err != nil {
			return err
		}
		for _, file := range files {
			if err := c.loadFile(file, patterns, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// splitLine returns the lowercase keyword and the arguments of a line.
func splitLine(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return "", nil
	}
	end := strings.IndexAny(line, " \t=")
	if end < 0 {
		return strings.ToLower(line), nil
	}
	key := strings.ToLower(line[:end])
	rest := strings.TrimLeft(line[end:], " \t")
	rest = strings.TrimLeft(strings.TrimPrefix(rest, "="), " \t")
	return key, splitArgs(rest)
}

// splitArgs splits on spaces, keeps "quoted strings" together and stops at
// a word starting with #.
func splitArgs(s string) []string {
	var args []string
	for {
		s = strings.TrimLeft(s, " \t")
		switch {
		case s == "" || s[0] == '#':
			return args
		case s[0] == '"':
			arg, rest, _ := strings.Cut(s[1:], `"`)
			args, s = append(args, arg), rest
		default:
			end := strings.IndexAny(s, " \t")
			if end < 0 {
				return append(args, s)
			}
			args, s = append(args, s[:end]), s[end:]
		}
	}
}

// Lookup returns the options for alias. For each option the first value
// found wins, except IdentityFile, which adds up. A nil Config is empty.
func (c *Config) Lookup(alias string) Host {
	var h Host
	if c == nil {
		return h
	}
	for _, b := range c.blocks {
		if !matches(b.patterns, alias) {
			continue
		}
		for _, o := range b.options {
			switch o.key {
			case "hostname":
				if h.HostName == "" {
					h.HostName = strings.ReplaceAll(o.value, "%h", alias)
				}
			case "user":
				if h.User == "" {
					h.User = o.value
				}
			case "port":
				if h.Port == 0 {
					h.Port, _ = strconv.Atoi(o.value)
				}
			case "identityfile":
				h.IdentityFiles = append(h.IdentityFiles, c.expandHome(o.value))
			case "proxyjump":
				if h.ProxyJump == "" {
					h.ProxyJump = o.value
				}
			}
		}
	}
	return h
}

// Known reports whether a Host line names alias literally, not by a
// wildcard.
func (c *Config) Known(alias string) bool {
	if c == nil {
		return false
	}
	for _, b := range c.blocks {
		for _, p := range b.patterns {
			if p == alias {
				return true
			}
		}
	}
	return false
}

func (c *Config) expandHome(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return filepath.Join(c.home, rest)
	}
	return p
}

// matches applies a Host line: at least one pattern must match and no
// negated one may.
func matches(patterns []string, alias string) bool {
	found := false
	for _, p := range patterns {
		negated, isNegated := strings.CutPrefix(p, "!")
		ok, err := path.Match(negated, alias)
		if err != nil || !ok {
			continue
		}
		if isNegated {
			return false
		}
		found = true
	}
	return found
}
