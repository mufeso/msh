// Package sshtest contains helpers for testing SSH code without real
// servers or real keys. Everything it creates is throwaway: generated
// per test and stored in t.TempDir().
package sshtest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

// Key is a generated ed25519 key pair.
type Key struct {
	Private ed25519.PrivateKey
	Signer  ssh.Signer
}

// Public returns the SSH public key.
func (k Key) Public() ssh.PublicKey { return k.Signer.PublicKey() }

// NewKey generates a fresh ed25519 key.
func NewKey(t testing.TB) Key {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return Key{Private: priv, Signer: signer}
}

// WriteKeyFile writes k in OpenSSH format to dir/name with mode 0600 and
// returns the path. A non-empty passphrase encrypts the file.
func WriteKeyFile(t testing.TB, dir, name string, k Key, passphrase string) string {
	t.Helper()
	var block *pem.Block
	var err error
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(k.Private, "msh-test")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(k.Private, "msh-test", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
