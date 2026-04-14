package app

import (
	"os"
	"strings"
	"testing"

	"example.com/msh/internal/sshtest"
)

func TestNoSecretsInAnyOutput(t *testing.T) {
	fleet := newTestFleet(t, 2, nil)
	keyData, err := os.ReadFile(fleet.identity)
	if err != nil {
		t.Fatal(err)
	}
	// Every line of the private key (except the PEM armor) is secret.
	var secrets []string
	for _, line := range strings.Split(string(keyData), "\n") {
		if line != "" && !strings.HasPrefix(line, "-----") {
			secrets = append(secrets, line)
		}
	}
	// An encrypted key with a passphrase, given as the identity file.
	encrypted := sshtest.WriteKeyFile(t, t.TempDir(), "enc", sshtest.NewKey(t), "Tr0ub4dor&3")
	secrets = append(secrets, "Tr0ub4dor&3")

	runs := [][]string{
		{"exec", "--all", "--command", "echo hi", "--verbose", "--log-format", "json", "--output", "json"},
		{"exec", "--all", "--command", "echo hi", "--verbose", "--output", "csv"},
		{"ping", "--all", "--verbose", "--output", "jsonl"},
		{"config", "show", "--verbose"},
	}
	for _, args := range runs {
		stdout, stderr, _ := fleet.run(t, args)
		checkNoSecrets(t, args, stdout+stderr, secrets)
	}
	stdout, stderr, _ := fleet.run(t, []string{"exec", "--all", "--command", "true", "--verbose"}, "--identity", encrypted)
	checkNoSecrets(t, []string{"encrypted identity"}, stdout+stderr, secrets)
}

func checkNoSecrets(t *testing.T, args []string, out string, secrets []string) {
	t.Helper()
	for _, s := range secrets {
		if strings.Contains(out, s) {
			t.Errorf("%v: output contains secret material %q", args, s[:min(len(s), 12)]+"...")
		}
	}
}
