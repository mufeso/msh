package config

import "testing"

// fakeEnv turns a map into a LookupFunc.
func fakeEnv(m map[string]string) LookupFunc {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestEnvLayer(t *testing.T) {
	l, err := EnvLayer(fakeEnv(map[string]string{
		EnvUser:   "envuser",
		EnvPort:   "2201",
		EnvOutput: "", // empty means unset
	}))
	if err != nil {
		t.Fatal(err)
	}
	if l.User == nil || *l.User != "envuser" || l.Port == nil || *l.Port != 2201 {
		t.Errorf("layer = %+v", l)
	}
	if l.Output != nil {
		t.Error("empty variable should be unset")
	}
}

func TestEnvLayerBadPort(t *testing.T) {
	if _, err := EnvLayer(fakeEnv(map[string]string{EnvPort: "ssh"})); err == nil {
		t.Error("want error for MSH_PORT=ssh")
	}
}

func TestPrecedenceFlagEnvFileDefault(t *testing.T) {
	file := Layer{User: ptr("file"), Port: ptr(1001), Output: ptr("json")}
	env := Layer{User: ptr("env"), Port: ptr(1002)}
	flags := Layer{User: ptr("flag")}
	s := Resolve(Source{SourceFile, file}, Source{SourceEnv, env}, Source{SourceFlag, flags})

	checks := []struct {
		key, gotValue, wantValue, wantSource string
	}{
		{KeyUser, s.User, "flag", SourceFlag},
		{KeyOutput, s.Output, "json", SourceFile},
		{KeyIdentityFile, s.IdentityFile, "", SourceDefault},
	}
	for _, c := range checks {
		if c.gotValue != c.wantValue || s.Sources[c.key] != c.wantSource {
			t.Errorf("%s = %q from %q, want %q from %q", c.key, c.gotValue, s.Sources[c.key], c.wantValue, c.wantSource)
		}
	}
	if s.Port != 1002 || s.Sources[KeyPort] != SourceEnv {
		t.Errorf("port = %d from %q, want 1002 from environment", s.Port, s.Sources[KeyPort])
	}
}

func TestDefaultConfigPath(t *testing.T) {
	if got := DefaultConfigPath(fakeEnv(map[string]string{"HOME": "/home/u"})); got != "/home/u/.config/msh/config.yaml" {
		t.Errorf("got %q", got)
	}
	if got := DefaultConfigPath(fakeEnv(map[string]string{"HOME": "/home/u", "XDG_CONFIG_HOME": "/cfg"})); got != "/cfg/msh/config.yaml" {
		t.Errorf("got %q", got)
	}
	if got := DefaultConfigPath(fakeEnv(nil)); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestEnvExecutionSettings(t *testing.T) {
	l, err := EnvLayer(fakeEnv(map[string]string{EnvMode: "sequential", EnvWorkers: "3", EnvOrdered: "true"}))
	if err != nil {
		t.Fatal(err)
	}
	s := Resolve(Source{SourceEnv, l})
	if s.Mode != ModeSequential || s.Workers != 3 || !s.Ordered {
		t.Errorf("settings = %+v", s)
	}
	if _, err := EnvLayer(fakeEnv(map[string]string{EnvOrdered: "maybe"})); err == nil {
		t.Error("MSH_ORDERED=maybe accepted")
	}
}

func TestExpandHome(t *testing.T) {
	tests := []struct{ in, want string }{
		{"~", "/home/u"},
		{"~/.ssh/known_hosts", "/home/u/.ssh/known_hosts"},
		{"/etc/msh.yaml", "/etc/msh.yaml"},
		{"~other/file", "~other/file"},
		{"relative/~/x", "relative/~/x"},
	}
	for _, tt := range tests {
		if got := ExpandHome(tt.in, "/home/u"); got != tt.want {
			t.Errorf("ExpandHome(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := ExpandHome("~/x", ""); got != "~/x" {
		t.Errorf("without home: %q", got)
	}
}

func TestSafeDefaults(t *testing.T) {
	d := Defaults()
	if d.KnownHostsFile == "" {
		t.Error("known_hosts_file must have a default: host keys are always verified")
	}
	s := d
	s.ExpandPaths("/home/u")
	if s.KnownHostsFile != "/home/u/.ssh/known_hosts" || s.Inventory != "/home/u/.config/msh/inventory.yaml" {
		t.Errorf("expanded = %+v", s)
	}
}
