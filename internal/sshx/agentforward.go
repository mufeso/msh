package sshx

import (
	"errors"

	"golang.org/x/crypto/ssh/agent"
)

// ForwardAgent lets the server of c reach the local agent. With a bastion
// only the target gets it. The session must ask too: see RequestAgent.
func ForwardAgent(c Client, a *Auth) error {
	if a.agent == nil {
		return errors.New("-A: no ssh-agent is running; start one and use ssh-add")
	}
	if chained, ok := c.(chainedClient); ok {
		c = chained.Client
	}
	adapter, ok := c.(clientAdapter)
	if !ok {
		return errors.New("-A: this connection can't forward an agent")
	}
	return agent.ForwardToAgent(adapter.c, a.agent)
}

// RequestAgent asks the server to offer the agent to this session.
func RequestAgent(s Session) error {
	adapter, ok := s.(sessionAdapter)
	if !ok {
		return errors.New("-A: this session can't forward an agent")
	}
	return agent.RequestAgentForwarding(adapter.Session)
}
