package pkgs

import (
	"strconv"
	"strings"
)

// dpkgOrder is the sort weight of a character in dpkg's algorithm: "~"
// before everything, then the end of the string and digits, then letters,
// then everything else.
func dpkgOrder(c byte) int {
	switch {
	case isDigit(c), c == 0:
		return 0
	case isAlpha(c):
		return int(c)
	case c == '~':
		return -1
	default:
		return int(c) + 256
	}
}

// verrevcmp compares two version strings like dpkg does.
func verrevcmp(a, b string) int {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		for (i < len(a) && !isDigit(a[i])) || (j < len(b) && !isDigit(b[j])) {
			x, y := dpkgOrder(at(a, i)), dpkgOrder(at(b, j))
			if x != y {
				return x - y
			}
			i++
			j++
		}
		for at(a, i) == '0' {
			i++
		}
		for at(b, j) == '0' {
			j++
		}
		firstDiff := 0
		for isDigit(at(a, i)) && isDigit(at(b, j)) {
			if firstDiff == 0 {
				firstDiff = int(a[i]) - int(b[j])
			}
			i++
			j++
		}
		if isDigit(at(a, i)) {
			return 1
		}
		if isDigit(at(b, j)) {
			return -1
		}
		if firstDiff != 0 {
			return firstDiff
		}
	}
	return 0
}

// parts is a version split into [epoch:]version[-release].
type parts struct {
	epoch    int
	hasEpoch bool
	version  string
	release  string
	hasRel   bool
}

func split(v string) parts {
	var p parts
	if e, rest, ok := strings.Cut(v, ":"); ok {
		if n, err := strconv.Atoi(e); err == nil {
			p.epoch, p.hasEpoch = n, true
			v = rest
		}
	}
	if i := strings.LastIndexByte(v, '-'); i >= 0 {
		p.release, p.hasRel = v[i+1:], true
		v = v[:i]
	}
	p.version = v
	return p
}

// segmentCompare returns the right algorithm for a package manager.
func segmentCompare(manager string) func(a, b string) int {
	if manager == DPKG {
		return verrevcmp
	}
	return rpmvercmp
}

// Compare orders two complete versions for a package manager. It returns
// a negative number, 0 or a positive number.
func Compare(manager, a, b string) int {
	x, y := split(a), split(b)
	if x.epoch != y.epoch {
		return x.epoch - y.epoch
	}
	cmp := segmentCompare(manager)
	if c := cmp(x.version, y.version); c != 0 {
		return c
	}
	return cmp(x.release, y.release)
}

// Satisfies checks an installed version against a requirement such as
// >= 3.0. Only the parts the requirement names are compared: an installed
// 1:3.0.7-16.el9 satisfies >= 3.0, because the requirement has no epoch
// and no release.
func Satisfies(manager, installed, op, want string) bool {
	x, y := split(installed), split(want)
	cmp := segmentCompare(manager)
	c := 0
	if y.hasEpoch {
		c = x.epoch - y.epoch
	}
	if c == 0 {
		c = cmp(x.version, y.version)
	}
	if c == 0 && y.hasRel {
		c = cmp(x.release, y.release)
	}
	switch op {
	case "=", "==":
		return c == 0
	case "!=":
		return c != 0
	case "<":
		return c < 0
	case "<=":
		return c <= 0
	case ">":
		return c > 0
	case ">=":
		return c >= 0
	default:
		return false
	}
}
