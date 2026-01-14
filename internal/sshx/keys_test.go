package sshx

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"example.com/msh/internal/sshtest"
)

func TestLoadSigner(t *testing.T) {
	dir := t.TempDir()
	key := sshtest.NewKey(t)
	path := sshtest.WriteKeyFile(t, dir, "id_ed25519", key, "")

	signer, err := LoadSigner(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(signer.PublicKey().Marshal(), key.Public().Marshal()) {
		t.Error("loaded a different key")
	}
}

func TestLoadSignerPassphraseProtected(t *testing.T) {
	dir := t.TempDir()
	path := sshtest.WriteKeyFile(t, dir, "id_enc", sshtest.NewKey(t), "correct horse")
	_, err := LoadSigner(path)
	if !errors.Is(err, ErrPassphraseProtected) {
		t.Fatalf("err = %v, want ErrPassphraseProtected", err)
	}
	if strings.Contains(err.Error(), "correct horse") {
		t.Error("error message leaks the passphrase")
	}
}

func TestLoadSignerRejectsOpenPermissions(t *testing.T) {
	dir := t.TempDir()
	path := sshtest.WriteKeyFile(t, dir, "id_open", sshtest.NewKey(t), "")
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSigner(path); err == nil || !strings.Contains(err.Error(), "too open") {
		t.Errorf("err = %v, want permissions error", err)
	}
}

func TestLoadSignerGarbage(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/garbage"
	if err := os.WriteFile(path, []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSigner(path); err == nil {
		t.Error("garbage accepted as key")
	}
}
