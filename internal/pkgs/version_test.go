package pkgs

import "testing"

func TestVerrevcmp(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0", 0},
		{"1.9", "1.10", -1},
		{"1.0~rc1", "1.0", -1},
		{"1.0", "1.0a", -1},
		{"1.0a", "1.0+", -1},
		{"1.0+b1", "1.0", 1},
		{"007", "7", 0},
		{"1.2.3", "1.2.3.0", -1},
		{"2.34", "2.4", 1},
	}
	for _, tt := range tests {
		if got := sign(verrevcmp(tt.a, tt.b)); got != tt.want {
			t.Errorf("verrevcmp(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
		if got := sign(verrevcmp(tt.b, tt.a)); got != -tt.want {
			t.Errorf("verrevcmp(%q, %q) = %d, want %d", tt.b, tt.a, got, -tt.want)
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		manager, a, b string
		want          int
	}{
		{RPM, "1:1.0-1", "2.0-1", 1},
		{RPM, "1.0-1.el9", "1.0-2.el9", -1},
		{RPM, "1.0-1", "1.0-1", 0},
		{DPKG, "1:1.0-1", "2.0-1", 1},
		{DPKG, "1.24.0-2ubuntu7", "1.24.0-2ubuntu7.1", -1},
		{DPKG, "3.0.13-0ubuntu3", "3.0.2-0ubuntu1", 1},
	}
	for _, tt := range tests {
		if got := sign(Compare(tt.manager, tt.a, tt.b)); got != tt.want {
			t.Errorf("Compare(%s, %q, %q) = %d, want %d", tt.manager, tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	tests := []struct {
		manager, installed, op, want string
		ok                           bool
	}{
		{RPM, "1:3.0.7-16.el9", ">=", "3.0", true},
		{RPM, "1:3.0.7-16.el9", ">=", "3.1", false},
		{RPM, "1:3.0.7-16.el9", ">=", "2:1.0", false},
		{RPM, "1:3.0.7-16.el9", "=", "3.0.7", true},
		{RPM, "1:3.0.7-16.el9", "=", "3.0.7-17.el9", false},
		{RPM, "1.9-1", "<", "1.10", true},
		{DPKG, "3.0.13-0ubuntu3", ">=", "3.0.2", true},
		{DPKG, "1:2.0-1", "!=", "2.0", false},
	}
	for _, tt := range tests {
		if got := Satisfies(tt.manager, tt.installed, tt.op, tt.want); got != tt.ok {
			t.Errorf("Satisfies(%s, %q, %s, %q) = %v", tt.manager, tt.installed, tt.op, tt.want, got)
		}
	}
}
