// Package shellquote quotes words for POSIX shells (sh, bash, dash, zsh).
//
// A remote SSH command is a single string that the remote user's login
// shell parses. To pass a word through unchanged, it is wrapped in single
// quotes, inside which a POSIX shell interprets nothing at all. A single
// quote itself is written as '\” (end quote, escaped quote, reopen).
package shellquote

import (
	"regexp"
	"strings"
)

// safeWord matches words that need no quoting.
var safeWord = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// Quote returns s as one shell word.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	if safeWord.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Join quotes each word and joins them with spaces.
func Join(words []string) string {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = Quote(w)
	}
	return strings.Join(quoted, " ")
}
