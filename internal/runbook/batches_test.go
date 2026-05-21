package runbook

import (
	"fmt"
	"strings"
	"testing"

	"example.com/msh/internal/inventory"
)

func fleet(n int) []inventory.Server {
	out := make([]inventory.Server, n)
	for i := range out {
		out[i] = inventory.Server{Name: fmt.Sprintf("s%d", i+1), Address: "192.0.2.1"}
	}
	return out
}

// shape renders batches as "s1,s2|s3".
func shape(batches [][]inventory.Server) string {
	var parts []string
	for _, b := range batches {
		var names []string
		for _, s := range b {
			names = append(names, s.Name)
		}
		parts = append(parts, strings.Join(names, ","))
	}
	return strings.Join(parts, "|")
}

func TestBatches(t *testing.T) {
	tests := []struct {
		n, size int
		want    string
	}{
		{5, 2, "s1,s2|s3,s4|s5"},
		{4, 2, "s1,s2|s3,s4"},
		{3, 0, "s1,s2,s3"},
		{2, 10, "s1,s2"},
		{0, 2, ""},
	}
	for _, tt := range tests {
		if got := shape(Batches(fleet(tt.n), tt.size)); got != tt.want {
			t.Errorf("Batches(%d, %d) = %q, want %q", tt.n, tt.size, got, tt.want)
		}
	}
}
