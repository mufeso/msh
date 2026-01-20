package config

import (
	"fmt"
	"strconv"
	"time"
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
	EnvConnectTimeout = "MSH_CONNECT_TIMEOUT"
	EnvCommandTimeout = "MSH_COMMAND_TIMEOUT"
	EnvTimeout        = "MSH_TIMEOUT"
	EnvMode           = "MSH_MODE"
	EnvWorkers        = "MSH_WORKERS"
	EnvOrdered        = "MSH_ORDERED"
	EnvBastion        = "MSH_BASTION"
)

// LookupFunc has the same signature as os.LookupEnv.
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
	l.Mode = str(EnvMode)
	l.Bastion = str(EnvBastion)

	var err error
	if l.Port, err = envInt(str, EnvPort); err != nil {
		return Layer{}, err
	}
	if l.ConnectTimeout, err = envDuration(str, EnvConnectTimeout); err != nil {
		return Layer{}, err
	}
	if l.CommandTimeout, err = envDuration(str, EnvCommandTimeout); err != nil {
		return Layer{}, err
	}
	if l.Timeout, err = envDuration(str, EnvTimeout); err != nil {
		return Layer{}, err
	}
	if l.Workers, err = envInt(str, EnvWorkers); err != nil {
		return Layer{}, err
	}
	if v := str(EnvOrdered); v != nil {
		b, perr := strconv.ParseBool(*v)
		if perr != nil {
			return Layer{}, fmt.Errorf("%s=%q: not a boolean (use true or false)", EnvOrdered, *v)
		}
		l.Ordered = &b
	}
	return l, nil
}

func envInt(str func(string) *string, name string) (*int, error) {
	v := str(name)
	if v == nil {
		return nil, nil
	}
	n, err := strconv.Atoi(*v)
	if err != nil {
		return nil, fmt.Errorf("%s=%q: not a number", name, *v)
	}
	return &n, nil
}

func envDuration(str func(string) *string, name string) (*Duration, error) {
	v := str(name)
	if v == nil {
		return nil, nil
	}
	d, err := time.ParseDuration(*v)
	if err != nil {
		return nil, fmt.Errorf("%s=%q: not a duration (use for example 30s)", name, *v)
	}
	dd := Duration(d)
	return &dd, nil
}
