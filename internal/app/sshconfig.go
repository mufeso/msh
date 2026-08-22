package app

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"example.com/msh/internal/config"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/sshconfig"
)

// userSSHConfig reads ~/.ssh/config. It is optional: without the file the
// result is nil, and a broken file is reported and ignored.
func (c *cli) userSSHConfig() *sshconfig.Config {
	home, _ := c.lookupEnv("HOME")
	if home == "" {
		return nil
	}
	cfg, err := sshconfig.Load(filepath.Join(home, ".ssh", "config"), home)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		c.errorf("ignoring SSH configuration: %v", err)
		return nil
	}
	return cfg
}

// fromCommandLine reports whether a setting came from a flag or the
// environment, which beat ~/.ssh/config.
func fromCommandLine(st *setup, key string) bool {
	source := st.settings.Sources[key]
	return source == config.SourceFlag || source == config.SourceEnv
}

// sshConfigResolver completes a new ad hoc server from ~/.ssh/config.
func (c *cli) sshConfigResolver(st *setup) func(*inventory.Server) error {
	cfg := c.userSSHConfig()
	if cfg == nil {
		return nil
	}
	return func(s *inventory.Server) error {
		h := cfg.Lookup(s.Address)
		s.Address = cmp.Or(h.HostName, s.Address)
		if s.User == "" && !fromCommandLine(st, config.KeyUser) {
			s.User = h.User
		}
		if s.Port == 0 && !fromCommandLine(st, config.KeyPort) {
			s.Port = h.Port
		}
		st.extraIdentities = append(st.extraIdentities, h.IdentityFiles...)
		return c.useProxyJump(st, cfg, s, h.ProxyJump)
	}
}

// useProxyJump routes s through a ProxyJump value. Several hops are an
// error: connecting directly instead would bypass the jump host.
func (c *cli) useProxyJump(st *setup, cfg *sshconfig.Config, s *inventory.Server, jump string) error {
	if jump == "" || jump == "none" {
		return nil
	}
	if strings.Contains(jump, ",") {
		return fmt.Errorf("ProxyJump with several hops is not supported: %s", jump)
	}
	// A plain word that is not in any file is a host name. The port makes
	// it a target, also for the pane of a tmux session.
	if _, known := st.inv.FindBastion(jump); !known && !cfg.Known(jump) && !strings.ContainsAny(jump, "@.:") {
		jump += ":" + strconv.Itoa(st.settings.Port)
	}
	s.Bastion = jump
	return c.addBastionTarget(st, jump)
}
