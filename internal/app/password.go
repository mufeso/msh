package app

import (
	"context"
	"fmt"
	"sync"

	"example.com/msh/internal/interactive"
)

// passwordPrompt asks once per login user, however many servers connect
// at the same time. The lock also keeps two prompts off the terminal.
type passwordPrompt struct {
	ctx  context.Context
	read interactive.ReadPassword

	mu     sync.Mutex
	byUser map[string]string
}

func (p *passwordPrompt) get(user, host string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if password, ok := p.byUser[user]; ok {
		return password, nil
	}
	password, err := p.read(p.ctx, fmt.Sprintf("%s@%s's password: ", user, host))
	if err != nil {
		return "", err
	}
	if p.byUser == nil {
		p.byUser = map[string]string{}
	}
	p.byUser[user] = password
	return password, nil
}
