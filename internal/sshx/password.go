package sshx

import "golang.org/x/crypto/ssh"

// PasswordFunc returns the password for user on host. SSH calls it only
// after the host key was verified and the keys were refused.
type PasswordFunc func(user, host string) (string, error)

// passwordMethods returns the password methods for one endpoint, tried
// after publickey. A nil cfg.Password means none.
func passwordMethods(cfg Config, ep Endpoint) []ssh.AuthMethod {
	if cfg.Password == nil {
		return nil
	}
	ask := func() (string, error) { return cfg.Password(ep.User, ep.Host) }
	return []ssh.AuthMethod{ssh.PasswordCallback(ask)}
}
