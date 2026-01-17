package app

import (
	"flag"
	"fmt"
	"path/filepath"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/exitcode"
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
}

// Close releases the agent connection.
func (s *sshStack) Close() { s.auth.Close() }

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
	return &sshStack{executor: &sshx.Executor{Connector: dialer}, auth: auth}, exitcode.OK
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
