package sshx

import (
	"bytes"
	"errors"
	"testing"

	"example.com/msh/internal/sshtest"
)

func TestDialAgent(t *testing.T) {
	key := sshtest.NewKey(t)
	socket := sshtest.StartAgent(t, key)

	ag, closer, err := DialAgent(socket)
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()

	signers, err := ag.Signers()
	if err != nil {
		t.Fatal(err)
	}
	if len(signers) != 1 || !bytes.Equal(signers[0].PublicKey().Marshal(), key.Public().Marshal()) {
		t.Errorf("agent signers = %d", len(signers))
	}
}

func TestDialAgentNoSocket(t *testing.T) {
	if _, _, err := DialAgent(""); !errors.Is(err, ErrNoAgent) {
		t.Errorf("err = %v, want ErrNoAgent", err)
	}
}
