package config

import (
	"fmt"
	"strconv"
)

// SourceEnv names the environment layer in Settings.Sources.
const SourceEnv = "environment"

// Environment variable names.
const (
	EnvConfig         = "MSH_CONFIG"
	EnvInventory      = "MSH_INVENTORY"
	EnvUser           = "MSH_USER"
	EnvPort           = "MSH_PORT"
	EnvIdentityFile   = "MSH_IDENTITY_FILE"
	EnvKnownHostsFile = "MSH_KNOWN_HOSTS_FILE"
	EnvOutput         = "MSH_OUTPUT"
)

// LookupFunc has the same signature as os.LookupEnv. Accepting a function
// instead of calling os.LookupEnv directly lets tests pass a fake
// environment.
type LookupFunc func(key string) (string, bool)

// EnvLayer builds a layer from environment variables. A variable that is
// set to an empty string is treated as unset.
func EnvLayer(lookup LookupFunc) (Layer, error) {
	var l Layer
	str := func(name string) *string {
		if v, ok := lookup(name); ok && v != "" {
			return &v
		}
		return nil
	}
	l.Inventory = str(EnvInventory)
	l.User = str(EnvUser)
	l.IdentityFile = str(EnvIdentityFile)
	l.KnownHostsFile = str(EnvKnownHostsFile)
	l.Output = str(EnvOutput)

	if v := str(EnvPort); v != nil {
		p, err := strconv.Atoi(*v)
		if err != nil {
			return Layer{}, fmt.Errorf("%s=%q: not a number", EnvPort, *v)
		}
		l.Port = &p
	}
	return l, nil
}
