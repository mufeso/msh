// Package expr implements the small expression language used by --where
// and runbook "when:" conditions:
//
//	os_family == "redhat" && cpus >= 4
//	!(disk_root_pct > 90) || name ~ "^db"
//
// An expression can only compare values; it can't run anything or change
// anything, so it is safe to evaluate from configuration files.
package expr

import (
	"fmt"
	"strings"
	"unicode"
)

// kind is a token type.
type kind int

const (
	tokEOF kind = iota
	tokIdent
	tokString
	tokNumber
	tokOp     // == != <= >= < > ~ !~
	tokAnd    // &&
	tokOr     // ||
	tokNot    // !
	tokLParen // (
	tokRParen // )
)

// token is one lexical element.
type token struct {
	kind kind
	text string
	pos  int // offset in runes, for error messages
}

// lex splits src into tokens, ending with tokEOF.
func lex(src string) ([]token, error) {
	runes := []rune(src)
	var out []token
	for i := 0; i < len(runes); {
		r := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		two := string([]rune{r, next})
		switch {
		case unicode.IsSpace(r):
			i++
		case two == "==" || two == "!=" || two == "<=" || two == ">=" || two == "!~":
			out = append(out, token{tokOp, two, i})
			i += 2
		case two == "&&":
			out = append(out, token{tokAnd, two, i})
			i += 2
		case two == "||":
			out = append(out, token{tokOr, two, i})
			i += 2
		case r == '<' || r == '>' || r == '~':
			out = append(out, token{tokOp, string(r), i})
			i++
		case r == '!':
			out = append(out, token{tokNot, "!", i})
			i++
		case r == '(':
			out = append(out, token{tokLParen, "(", i})
			i++
		case r == ')':
			out = append(out, token{tokRParen, ")", i})
			i++
		case r == '"' || r == '\'':
			text, end, err := lexString(runes, i)
			if err != nil {
				return nil, err
			}
			out = append(out, token{tokString, text, i})
			i = end
		case unicode.IsDigit(r):
			end := i
			for end < len(runes) && (unicode.IsDigit(runes[end]) || runes[end] == '.') {
				end++
			}
			out = append(out, token{tokNumber, string(runes[i:end]), i})
			i = end
		case isIdentRune(r):
			end := i
			for end < len(runes) && isIdentRune(runes[end]) {
				end++
			}
			out = append(out, token{tokIdent, string(runes[i:end]), i})
			i = end
		default:
			return nil, fmt.Errorf("position %d: unexpected character %q", i+1, string(r))
		}
	}
	return append(out, token{tokEOF, "", len(runes)}), nil
}

// isIdentRune accepts letters, digits, "_", "." and "-" inside names.
func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '-'
}

// lexString reads a quoted string starting at runes[start]. A backslash
// escapes the next character.
func lexString(runes []rune, start int) (text string, end int, err error) {
	quote := runes[start]
	var b strings.Builder
	for i := start + 1; i < len(runes); i++ {
		switch runes[i] {
		case '\\':
			i++
			if i < len(runes) {
				b.WriteRune(runes[i])
			}
		case quote:
			return b.String(), i + 1, nil
		default:
			b.WriteRune(runes[i])
		}
	}
	return "", 0, fmt.Errorf("position %d: unterminated string", start+1)
}
