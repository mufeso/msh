package expr

import "testing"

func TestEval(t *testing.T) {
	env := map[string]string{
		"os_family": "redhat", "cpus": "8", "memory_mb": "16000", "disk": "91",
		"name": "db01", "version": "9.4", "flag": "true", "off": "false", "zero": "0", "empty": "",
	}
	lookup := func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}
	tests := map[string]bool{
		`os_family == "redhat"`:              true,
		`os_family != "redhat"`:              false,
		`cpus >= 4`:                          true,
		`cpus > 10`:                          false,
		`cpus < 10`:                          true, // numeric: as strings "8" > "10"
		`version == 9.4`:                     true,
		`name ~ "^db"`:                       true,
		`name !~ "^db"`:                      false,
		`os_family == "redhat" && cpus >= 4`: true,
		`os_family == "debian" || cpus >= 4`: true,
		`!(disk > 90)`:                       false,
		`flag`:                               true,
		`off`:                                false,
		`zero`:                               false,
		`empty`:                              false,
		`missing`:                            false,
		`missing == ""`:                      true,
		`!missing && cpus == 8`:              true,
	}
	for in, want := range tests {
		e, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if got := e.Eval(lookup); got != want {
			t.Errorf("%s = %v, want %v", in, got, want)
		}
	}
}
