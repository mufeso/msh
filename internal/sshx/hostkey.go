package sshx

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyError reports a host key that could not be verified.
type HostKeyError struct {
	Host        string // host:port as presented to the callback
	Fingerprint string // SHA256 fingerprint of the key the server sent
	Unknown     bool   // true: host not in known_hosts; false: key CHANGED
	File        string // known_hosts file that was consulted
}

func (e *HostKeyError) Error() string {
	if e.Unknown {
		return fmt.Sprintf("host key for %s is not in %s (server key %s); verify the fingerprint out of band, then add it, e.g. with ssh-keyscan",
			e.Host, e.File, e.Fingerprint)
	}
	return fmt.Sprintf("HOST KEY MISMATCH for %s: the server sent %s, which differs from %s; this may be a man-in-the-middle attack",
		e.Host, e.Fingerprint, e.File)
}

// HostKeyCallback returns the callback that verifies server host keys.
//
// With cfg.InsecureIgnoreHostKey every key is accepted. That disables
// protection against man-in-the-middle attacks; the caller must warn the
// user prominently.
func HostKeyCallback(cfg Config) (ssh.HostKeyCallback, error) {
	if cfg.InsecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	if cfg.KnownHostsFile == "" {
		return nil, errors.New("no known_hosts file configured")
	}
	check, err := knownhosts.New(cfg.KnownHostsFile)
	if err != nil {
		return nil, fmt.Errorf("reading known_hosts: %w", err)
	}

	// The returned function wraps check: it calls it, then translates its
	// error into our own type. This is a closure over check and cfg.
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := check(hostname, remote, key)
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			return &HostKeyError{
				Host:        hostname,
				Fingerprint: ssh.FingerprintSHA256(key),
				Unknown:     len(keyErr.Want) == 0, // no key known at all
				File:        cfg.KnownHostsFile,
			}
		}
		return err
	}, nil
}

// placeholderKey is never a real host key. It is used to ask the
// known_hosts callback which keys it expects for a host.
var placeholderKey, _ = ssh.NewPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))

// HostKeyAlgorithms returns the host key algorithms recorded in
// known_hosts for addr, or nil if none are known.
//
// Without this, a server offering an RSA and an ed25519 key might pick the
// type we do not have recorded, and verification would fail as if the key
// had changed.
func HostKeyAlgorithms(cfg Config, addr string) []string {
	if cfg.InsecureIgnoreHostKey || cfg.KnownHostsFile == "" {
		return nil
	}
	check, err := knownhosts.New(cfg.KnownHostsFile)
	if err != nil {
		return nil
	}
	var keyErr *knownhosts.KeyError
	if err := check(addr, &net.TCPAddr{}, placeholderKey); !errors.As(err, &keyErr) {
		return nil
	}
	var algos []string
	seen := map[string]bool{}
	for _, known := range keyErr.Want {
		for _, a := range algorithmsForKeyType(known.Key.Type()) {
			if !seen[a] {
				seen[a] = true
				algos = append(algos, a)
			}
		}
	}
	return algos
}

// algorithmsForKeyType maps a key type to the signature algorithms that
// can be negotiated for it. RSA keys sign with SHA-2 by default today.
func algorithmsForKeyType(keyType string) []string {
	if keyType == ssh.KeyAlgoRSA {
		return []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA}
	}
	return []string{keyType}
}
