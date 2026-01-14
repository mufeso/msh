package sshx

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// ErrPassphraseProtected means a private key file is encrypted. msh never
// asks for or stores passphrases: load the key into ssh-agent instead
// (ssh-add FILE) and msh will use the agent.
var ErrPassphraseProtected = errors.New("private key is passphrase-protected; add it to ssh-agent with ssh-add")

// LoadSigner reads and parses a private key file.
//
// Like OpenSSH, it refuses key files that other users can read. Error
// messages contain the path but never any key material.
func LoadSigner(path string) (ssh.Signer, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return nil, fmt.Errorf("%s: permissions %#o are too open; run: chmod 600 %s", path, perm, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			return nil, fmt.Errorf("%s: %w", path, ErrPassphraseProtected)
		}
		return nil, fmt.Errorf("%s: cannot parse private key: %w", path, err)
	}
	return signer, nil
}
