package pkgs

import "strings"

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// at returns s[i], or 0 past the end, which keeps the scanners simple.
func at(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return 0
}

// rpmvercmp compares two version strings like rpm does. It returns a
// negative number, 0 or a positive number.
//
// Both strings are split into segments of digits or letters. Numbers
// compare as numbers and beat letters. "~" sorts before anything, even the
// end of the string (1.0~rc1 < 1.0); "^" sorts after the end but before
// other text (1.0 < 1.0^post < 1.0.1).
func rpmvercmp(a, b string) int {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		for i < len(a) && !isDigit(a[i]) && !isAlpha(a[i]) && a[i] != '~' && a[i] != '^' {
			i++
		}
		for j < len(b) && !isDigit(b[j]) && !isAlpha(b[j]) && b[j] != '~' && b[j] != '^' {
			j++
		}
		if at(a, i) == '~' || at(b, j) == '~' {
			if at(a, i) != '~' {
				return 1
			}
			if at(b, j) != '~' {
				return -1
			}
			i++
			j++
			continue
		}
		if at(a, i) == '^' || at(b, j) == '^' {
			switch {
			case i >= len(a):
				return -1
			case j >= len(b):
				return 1
			case a[i] != '^':
				return 1
			case b[j] != '^':
				return -1
			}
			i++
			j++
			continue
		}
		if i >= len(a) || j >= len(b) {
			break
		}

		numeric := isDigit(a[i])
		match := isAlpha
		if numeric {
			match = isDigit
		}
		si, sj := i, j
		for i < len(a) && match(a[i]) {
			i++
		}
		for j < len(b) && match(b[j]) {
			j++
		}
		if sj == j { // different kinds of segment: numbers beat letters
			if numeric {
				return 1
			}
			return -1
		}
		segA, segB := a[si:i], b[sj:j]
		if numeric {
			// Compare numbers of any length: no zeros, then the longer one
			// is bigger, and equal lengths compare as text.
			segA, segB = strings.TrimLeft(segA, "0"), strings.TrimLeft(segB, "0")
			if len(segA) != len(segB) {
				return len(segA) - len(segB)
			}
		}
		if c := strings.Compare(segA, segB); c != 0 {
			return c
		}
	}
	switch {
	case i >= len(a) && j >= len(b):
		return 0
	case i < len(a):
		return 1
	default:
		return -1
	}
}
