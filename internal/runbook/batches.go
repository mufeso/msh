package runbook

import "example.com/msh/internal/inventory"

// Batches splits servers into consecutive groups of at most size servers.
// A size of 0 or less means one batch with everything.
func Batches(servers []inventory.Server, size int) [][]inventory.Server {
	if len(servers) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(servers)
	}
	var out [][]inventory.Server
	for start := 0; start < len(servers); start += size {
		end := min(start+size, len(servers))
		// The third index limits capacity: an append to one batch can
		// never write into the next.
		out = append(out, servers[start:end:end])
	}
	return out
}
