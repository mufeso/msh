package sshx

import (
	"errors"
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh/agent"
)

// ErrNoAgent means no agent socket is configured.
var ErrNoAgent = errors.New("SSH_AUTH_SOCK is not set; no ssh-agent available")

// DialAgent connects to the ssh-agent listening on socket. The returned
// io.Closer closes the connection; callers must call it when done.
//
// The agent performs signatures itself: private keys and passphrases
// never leave it.
func DialAgent(socket string) (agent.ExtendedAgent, io.Closer, error) {
	if socket == "" {
		return nil, nil, ErrNoAgent
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to ssh-agent: %w", err)
	}
	return agent.NewClient(conn), conn, nil
}
