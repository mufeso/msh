package sshx

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"example.com/msh/internal/sshtest"
)

func TestNewAuthExplicitKey(t *testing.T) {
	dir := t.TempDir()
	path := sshtest.WriteKeyFile(t, dir, "id", sshtest.NewKey(t), "")
	a, err := NewAuth(Config{IdentityFile: path})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if len(a.fileSigners) != 1 {
		t.Errorf("file signers = %d", len(a.fileSigners))
	}
}

func TestNewAuthEncryptedKeyNeedsAgent(t *testing.T) {
	dir := t.TempDir()
	key := sshtest.NewKey(t)
	path := sshtest.WriteKeyFile(t, dir, "id", key, "secret")

	if _, err := NewAuth(Config{IdentityFile: path}); !errors.Is(err, ErrPassphraseProtected) {
		t.Fatalf("without agent: err = %v", err)
	}

	socket := sshtest.StartAgent(t, key)
	a, err := NewAuth(Config{IdentityFile: path, AgentSocket: socket})
	if err != nil {
		t.Fatalf("with agent: %v", err)
	}
	defer a.Close()
	if len(a.Notes()) == 0 {
		t.Error("expected a note explaining the skipped encrypted file")
	}
}

func TestNewAuthDefaultFilesAreOptional(t *testing.T) {
	dir := t.TempDir()
	present := sshtest.WriteKeyFile(t, dir, "id_ed25519", sshtest.NewKey(t), "")
	a, err := NewAuth(Config{DefaultIdentityFiles: []string{filepath.Join(dir, "id_missing"), present}})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
}

func TestNewAuthNoKeys(t *testing.T) {
	if _, err := NewAuth(Config{DefaultIdentityFiles: []string{t.TempDir() + "/none"}}); err == nil {
		t.Error("NewAuth succeeded without any key")
	}
}

func TestClientConfig(t *testing.T) {
	dir := t.TempDir()
	a, err := NewAuth(Config{IdentityFile: sshtest.WriteKeyFile(t, dir, "id", sshtest.NewKey(t), "")})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	cfg := Config{ConnectTimeout: 7 * time.Second, InsecureIgnoreHostKey: true}
	cb, _ := HostKeyCallback(cfg)
	cc := ClientConfig(cfg, Endpoint{User: "ops", Host: "192.0.2.1", Port: 22}, a, cb)
	if cc.User != "ops" || cc.Timeout != 7*time.Second || len(cc.Auth) != 1 || cc.HostKeyCallback == nil {
		t.Errorf("client config = %+v", cc)
	}
}
