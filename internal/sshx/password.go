package sshx

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

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
	challenge := func(_, _ string, questions []string, echos []bool) ([]string, error) {
		return answerChallenge(questions, echos, ask)
	}
	return []ssh.AuthMethod{ssh.PasswordCallback(ask), ssh.KeyboardInteractive(challenge)}
}

// answerChallenge answers keyboard-interactive questions. Only hidden
// questions that ask for a password get it: the password must not end up
// in a field for something else.
func answerChallenge(questions []string, echos []bool, ask func() (string, error)) ([]string, error) {
	answers := make([]string, len(questions))
	for i, q := range questions {
		if echos[i] || !strings.Contains(strings.ToLower(q), "password") {
			return nil, fmt.Errorf("the server asks for something other than a password: %q", q)
		}
		password, err := ask()
		if err != nil {
			return nil, err
		}
		answers[i] = password
	}
	return answers, nil
}
