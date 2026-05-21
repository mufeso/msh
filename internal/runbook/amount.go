package runbook

import (
	"fmt"
	"strconv"
	"strings"
)

// Amount is a count of servers: an absolute number or a percentage of the
// selection.
type Amount struct {
	Value   int
	Percent bool
}

// ParseAmount parses "2" or "25%".
func ParseAmount(s string) (Amount, error) {
	text, percent := strings.CutSuffix(strings.TrimSpace(s), "%")
	v, err := strconv.Atoi(text)
	if err != nil || v < 0 || (percent && v > 100) {
		return Amount{}, fmt.Errorf("%q is not a number or a percentage from 0%% to 100%%", s)
	}
	return Amount{Value: v, Percent: percent}, nil
}

// Of resolves the amount for a selection of total servers. Percentages
// round up: 25% of 10 servers is 3.
func (a Amount) Of(total int) int {
	if !a.Percent {
		return a.Value
	}
	return (a.Value*total + 99) / 100
}
