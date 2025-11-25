package app

// levenshtein returns the minimum number of single-character insertions,
// deletions or substitutions needed to turn a into b.
//
// The strings are converted to []rune so that multi-byte UTF-8 characters
// count as one character each.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)

	// prev[j] holds the distance between the first i-1 runes of a and the
	// first j runes of b. Only two rows are needed at any time.
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

// suggest returns the candidate closest to input, if it is close enough to
// be a plausible typo. ok is false when nothing is close.
func suggest(input string, candidates []string) (best string, ok bool) {
	const maxDistance = 2
	bestDist := maxDistance + 1
	for _, c := range candidates {
		if d := levenshtein(input, c); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best, bestDist <= maxDistance
}

// commandNames returns the names of all commands.
func commandNames() []string {
	var names []string
	for _, cmd := range commands() {
		names = append(names, cmd.name)
	}
	return names
}
