package app

import (
	"flag"
	"fmt"
	"path/filepath"

	"example.com/msh/internal/config"
	"example.com/msh/internal/errs"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/sshx"
)

// insecureWarning is printed whenever host key verification is disabled.
const insecureWarning = `WARNING: host key verification is DISABLED (--insecure-ignore-host-key).
WARNING: anyone between you and the servers can impersonate them and read
WARNING: everything you send. Use this only on isolated test networks.`

// remoteOptions are flags for commands that connect to servers.
type remoteOptions struct {
	insecureIgnoreHostKey bool
}

func (o *remoteOptions) register(fs *flag.FlagSet) {
	fs.BoolVar(&o.insecureIgnoreHostKey, "insecure-ignore-host-key", false,
		"UNSAFE: do not verify server host keys")
}

// sshStack is the SSH machinery for one run.
type sshStack struct {
	executor *sshx.Executor
	auth     *sshx.Auth
	dialer   *sshx.Dialer
}

// Close releases agent connections and bastion credentials.
func (s *sshStack) Close() {
	s.dialer.Close()
	s.auth.Close()
}

// newSSH builds the SSH executor from the resolved settings. On failure
// it prints the error and returns nil and the exit code.
func (c *cli) newSSH(opts *commonOptions, ro *remoteOptions, st *setup) (*sshStack, exitcode.Code) {
	cfg := c.sshConfig(ro, st)
	if cfg.InsecureIgnoreHostKey {
		fmt.Fprintln(c.stderr, insecureWarning)
	}
	auth, err := sshx.NewAuth(cfg)
	if err != nil {
		c.errorf("%v", err)
		return nil, exitcode.Connection
	}
	for _, note := range auth.Notes() {
		c.verbosef(opts, "ssh: %s", note)
	}
	dialer, err := sshx.NewDialer(cfg, auth)
	if err != nil {
		auth.Close()
		c.errorf("%v", err)
		return nil, errs.ExitCode(errs.KindOf(err))
	}
	if st.inv != nil {
		home, _ := c.lookupEnv("HOME")
		for _, b := range st.inv.Bastions {
			err := dialer.AddBastion(sshx.BastionSpec{
				Name:           b.Name,
				Endpoint:       c.bastionEndpoint(b, st),
				IdentityFile:   config.ExpandHome(b.IdentityFile, home),
				KnownHostsFile: config.ExpandHome(b.KnownHostsFile, home),
			})
			if err != nil {
				dialer.Close()
				auth.Close()
				c.errorf("%v", err)
				return nil, errs.ExitCode(errs.KindOf(err))
			}
		}
	}
	return &sshStack{executor: &sshx.Executor{Connector: dialer}, auth: auth, dialer: dialer}, exitcode.OK
}

// bastionEndpoint completes a bastion's connection details: its own user
// and port, else the resolved settings, else the local user.
func (c *cli) bastionEndpoint(b inventory.Bastion, st *setup) sshx.Endpoint {
	ep := sshx.Endpoint{Name: b.Name, Host: b.Address, Port: b.Port, User: b.User}
	if ep.Port == 0 {
		ep.Port = st.settings.Port
	}
	if ep.User == "" {
		ep.User = st.settings.User
	}
	if ep.User == "" {
		ep.User = currentUser()
	}
	return ep
}

// sshConfig translates settings into sshx.Config.
func (c *cli) sshConfig(ro *remoteOptions, st *setup) sshx.Config {
	home, _ := c.lookupEnv("HOME")
	agentSocket, _ := c.lookupEnv("SSH_AUTH_SOCK")
	var defaults []string
	if home != "" {
		for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
			defaults = append(defaults, filepath.Join(home, ".ssh", name))
		}
	}
	return sshx.Config{
		IdentityFile:          st.settings.IdentityFile,
		DefaultIdentityFiles:  defaults,
		KnownHostsFile:        st.settings.KnownHostsFile,
		InsecureIgnoreHostKey: ro.insecureIgnoreHostKey,
		ConnectTimeout:        st.settings.ConnectTimeout,
		AgentSocket:           agentSocket,
	}
}
