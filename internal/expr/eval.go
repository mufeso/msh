package expr

import "strconv"

// Lookup returns the value of a name.
type Lookup func(name string) (string, bool)

// Eval reports whether the expression is true for the given names.
func (e *Expr) Eval(lookup Lookup) bool {
	return truth(eval(e.root, lookup))
}

// eval returns the value of a node. Boolean results are "true" or "".
func eval(n node, lookup Lookup) string {
	switch n := n.(type) {
	case identNode:
		v, _ := lookup(n.name)
		return v
	case literalNode:
		return n.value
	case notNode:
		return boolean(!truth(eval(n.inner, lookup)))
	case andNode: // the right side only runs if needed
		return boolean(truth(eval(n.left, lookup)) && truth(eval(n.right, lookup)))
	case orNode:
		return boolean(truth(eval(n.left, lookup)) || truth(eval(n.right, lookup)))
	case matchNode:
		return boolean(n.re.MatchString(eval(n.left, lookup)) != n.negate)
	case compareNode:
		return boolean(compare(n.op, eval(n.left, lookup), eval(n.right, lookup)))
	default:
		return ""
	}
}

func boolean(b bool) string {
	if b {
		return "true"
	}
	return ""
}

// truth is the boolean meaning of a value.
func truth(v string) bool {
	return v != "" && v != "0" && v != "false"
}

// compare compares two values, as numbers if both are numbers.
func compare(op, a, b string) bool {
	x, errA := strconv.ParseFloat(a, 64)
	y, errB := strconv.ParseFloat(b, 64)
	order := 0
	if errA == nil && errB == nil {
		switch {
		case x < y:
			order = -1
		case x > y:
			order = 1
		}
	} else {
		switch {
		case a < b:
			order = -1
		case a > b:
			order = 1
		}
	}
	switch op {
	case "==":
		return order == 0
	case "!=":
		return order != 0
	case "<":
		return order < 0
	case "<=":
		return order <= 0
	case ">":
		return order > 0
	case ">=":
		return order >= 0
	default:
		return false
	}
}
