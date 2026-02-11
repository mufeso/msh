package task

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// placeholderRE matches {{name}} or {{1}}, with optional spaces.
var placeholderRE = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// Values maps argument names to their values.
type Values map[string]string

// argName resolves a placeholder reference ("pattern" or "1") to an
// argument name.
func (t *Task) argName(ref string) (string, error) {
	if n, err := strconv.Atoi(ref); err == nil {
		if n < 1 || n > len(t.Arguments) {
			return "", fmt.Errorf("{{%s}}: the task has %d argument(s)", ref, len(t.Arguments))
		}
		return t.Arguments[n-1].Name, nil
	}
	for _, a := range t.Arguments {
		if a.Name == ref {
			return ref, nil
		}
	}
	return "", fmt.Errorf("{{%s}}: no argument with that name", ref)
}

// expand replaces each placeholder in tmpl with quote(value).
func (t *Task) expand(tmpl string, values Values, quote func(string) string) (string, error) {
	var b strings.Builder
	last := 0
	for _, m := range placeholderRE.FindAllStringSubmatchIndex(tmpl, -1) {
		// m[0]:m[1] is the whole match, m[2]:m[3] the reference.
		name, err := t.argName(tmpl[m[2]:m[3]])
		if err != nil {
			return "", err
		}
		b.WriteString(tmpl[last:m[0]])
		b.WriteString(quote(values[name]))
		last = m[1]
	}
	b.WriteString(tmpl[last:])
	return b.String(), nil
}

// validatePlaceholders checks that every placeholder in the task refers
// to a defined argument and that script tasks contain none.
func (t *Task) validatePlaceholders() error {
	templates := append([]string{t.Command}, t.Argv...)
	for _, tmpl := range templates {
		for _, m := range placeholderRE.FindAllStringSubmatch(tmpl, -1) {
			if _, err := t.argName(m[1]); err != nil {
				return err
			}
		}
	}
	if placeholderRE.MatchString(t.Script) {
		return fmt.Errorf("scripts receive arguments as positional parameters (\"$1\"); {{...}} placeholders are not substituted in scripts")
	}
	seen := map[string]bool{}
	for _, a := range t.Arguments {
		if !namePattern.MatchString(a.Name) {
			return fmt.Errorf("argument name %q must match %s", a.Name, namePattern)
		}
		if seen[a.Name] {
			return fmt.Errorf("duplicate argument %q", a.Name)
		}
		seen[a.Name] = true
	}
	return nil
}

// Bind assigns positional values to arguments in definition order.
func (t *Task) Bind(positional []string) (Values, error) {
	if len(positional) > len(t.Arguments) {
		return nil, fmt.Errorf("task %q takes %d argument(s), got %d", t.Name, len(t.Arguments), len(positional))
	}
	values := Values{}
	for i, v := range positional {
		values[t.Arguments[i].Name] = v
	}
	return values, nil
}
