package pkgs

import "testing"

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

func TestRpmvercmp(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0", 0},
		{"1.0", "2.0", -1},
		{"2.0.1", "2.0", 1},
		{"1.9", "1.10", -1},
		{"1.05", "1.5", 0},
		{"5.5p1", "5.5p2", -1},
		{"5.5p10", "5.5p1", 1},
		{"10xyz", "10.1xyz", -1},
		{"xyz.4", "8", -1},
		{"1.0~rc1", "1.0", -1},
		{"1.0~rc1", "1.0~rc2", -1},
		{"1.0~rc1~git123", "1.0~rc1", -1},
		{"1.0^", "1.0", 1},
		{"1.0^git1", "1.0", 1},
		{"1.0^git1", "1.01", -1},
		{"1.0^20160101", "1.0.1", -1},
		{"99999999999999999999.1", "99999999999999999998.9", 1},
		{"1.0.", "1.0", 0},
	}
	for _, tt := range tests {
		if got := sign(rpmvercmp(tt.a, tt.b)); got != tt.want {
			t.Errorf("rpmvercmp(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
		if got := sign(rpmvercmp(tt.b, tt.a)); got != -tt.want {
			t.Errorf("rpmvercmp(%q, %q) = %d, want %d", tt.b, tt.a, got, -tt.want)
		}
	}
}
