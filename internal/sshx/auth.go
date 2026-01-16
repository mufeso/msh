package sshx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Auth holds the authentication material for one run. Create it once
// with NewAuth, share it across connections, and Close it at the end.
type Auth struct {
	fileSigners []ssh.Signer
	agent       agent.ExtendedAgent
	agentCloser io.Closer
	notes       []string // why some keys were skipped; safe to print
}

// NewAuth loads keys according to cfg:
//
//   - cfg.IdentityFile, if set, must load, unless it is passphrase-protected
//     and an agent is available (the agent is expected to hold it).
//   - Otherwise cfg.DefaultIdentityFiles are tried; missing or encrypted
//     ones are skipped.
//   - The agent at cfg.AgentSocket is used when reachable.
//
// It fails if no key is available at all.
func NewAuth(cfg Config) (*Auth, error) {
	a := &Auth{}
	if ag, closer, err := DialAgent(cfg.AgentSocket); err == nil {
		a.agent, a.agentCloser = ag, closer
	} else if !errors.Is(err, ErrNoAgent) {
		a.notes = append(a.notes, err.Error())
	}

	if cfg.IdentityFile != "" {
		signer, err := LoadSigner(cfg.IdentityFile)
		switch {
		case err == nil:
			a.fileSigners = append(a.fileSigners, signer)
		case errors.Is(err, ErrPassphraseProtected) && a.agent != nil:
			a.notes = append(a.notes, err.Error()+" (relying on the agent)")
		default:
			a.Close()
			return nil, err
		}
	} else {
		for _, path := range cfg.DefaultIdentityFiles {
			signer, err := LoadSigner(path)
			switch {
			case err == nil:
				a.fileSigners = append(a.fileSigners, signer)
			case errors.Is(err, fs.ErrNotExist):
				// Optional default key: skip silently.
			default:
				a.notes = append(a.notes, err.Error())
			}
		}
	}

	if len(a.fileSigners) == 0 && a.agentKeyCount() == 0 {
		a.Close()
		return nil, fmt.Errorf("no usable SSH keys: start ssh-agent and add a key with ssh-add, or set identity_file")
	}
	return a, nil
}

// Notes explains skipped keys. The messages never contain key material.
func (a *Auth) Notes() []string { return a.notes }

func (a *Auth) agentKeyCount() int {
	if a.agent == nil {
		return 0
	}
	keys, err := a.agent.List()
	if err != nil {
		return 0
	}
	return len(keys)
}

// Method returns the publickey authentication method. The callback runs
// for each connection, so keys added to the agent later are seen.
func (a *Auth) Method() ssh.AuthMethod {
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		var signers []ssh.Signer
		if a.agent != nil {
			if s, err := a.agent.Signers(); err == nil {
				signers = append(signers, s...)
			}
		}
		return append(signers, a.fileSigners...), nil
	})
}

// Close releases the agent connection. It is safe to call more than once.
func (a *Auth) Close() error {
	if a.agentCloser == nil {
		return nil
	}
	err := a.agentCloser.Close()
	a.agentCloser = nil
	return err
}

// ClientConfig builds the SSH client configuration for one endpoint.
func ClientConfig(cfg Config, ep Endpoint, auth *Auth, hostKeys ssh.HostKeyCallback) *ssh.ClientConfig {
	cc := baseClientConfig(cfg, ep)
	cc.Auth = []ssh.AuthMethod{auth.Method()}
	cc.HostKeyCallback = hostKeys
	cc.HostKeyAlgorithms = HostKeyAlgorithms(cfg, ep.Addr())
	return cc
}
