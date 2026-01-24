package sshx

import (
	"context"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/sshtest"
)

// TestBastionFailureModes covers every hop/kind combination. In each case
// the target server must not run anything.
func TestBastionFailureModes(t *testing.T) {
	type setup struct {
		bastionAuthorized bool
		bastionKnown      bool
		bastionDown       bool
		targetAuthorized  bool
		targetKnown       bool
		targetDown        bool
	}
	ok := setup{true, true, false, true, true, false}
	tests := []struct {
		name     string
		modify   func(*setup)
		wantKind errs.Kind
		wantRole errs.Role
		wantVia  bool
	}{
		{"bastion connection", func(s *setup) { s.bastionDown = true }, errs.KindTCP, errs.RoleBastion, false},
		{"bastion authentication", func(s *setup) { s.bastionAuthorized = false }, errs.KindAuth, errs.RoleBastion, false},
		{"bastion host key", func(s *setup) { s.bastionKnown = false }, errs.KindHostKey, errs.RoleBastion, false},
		{"target connection", func(s *setup) { s.targetDown = true }, errs.KindTCP, errs.RoleTarget, true},
		{"target authentication", func(s *setup) { s.targetAuthorized = false }, errs.KindAuth, errs.RoleTarget, true},
		{"target host key", func(s *setup) { s.targetKnown = false }, errs.KindHostKey, errs.RoleTarget, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ok
			tt.modify(&s)
			key := sshtest.NewKey(t)
			stranger := sshtest.NewKey(t).Public()
			pick := func(authorized bool) []ssh.PublicKey {
				if authorized {
					return []ssh.PublicKey{key.Public()}
				}
				return []ssh.PublicKey{stranger}
			}
			target := sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: pick(s.targetAuthorized)})
			bastion := sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: pick(s.bastionAuthorized), AllowForwarding: true})

			dir := t.TempDir()
			var known []string
			if s.bastionKnown {
				known = append(known, bastion.KnownHostsLine())
			}
			if s.targetKnown {
				known = append(known, target.KnownHostsLine())
			}
			cfg := Config{
				IdentityFile:   sshtest.WriteKeyFile(t, dir, "id", key, ""),
				KnownHostsFile: sshtest.WriteKnownHosts(t, dir, known...),
				ConnectTimeout: 5e9,
			}
			auth, err := NewAuth(cfg)
			if err != nil {
				t.Fatal(err)
			}
			defer auth.Close()
			d, err := NewDialer(cfg, auth)
			if err != nil {
				t.Fatal(err)
			}

			bep := endpointOf("jump", bastion)
			tsrv := (&testEnv{server: target}).server0()
			if s.bastionDown {
				bastion.Close()
			}
			if s.targetDown {
				ln, _ := net.Listen("tcp", "127.0.0.1:0")
				tsrv.Port = ln.Addr().(*net.TCPAddr).Port
				ln.Close()
			}
			d.AddBastion("jump", bep)
			tsrv.Bastion = "jump"

			r := (&Executor{Connector: d}).Run(context.Background(), tsrv, executor.Request{Command: "true"})
			if r.Status != executor.StatusError {
				t.Fatalf("status = %s, err = %v", r.Status, r.Err)
			}
			if errs.KindOf(r.Err) != tt.wantKind || errs.RoleOf(r.Err) != tt.wantRole {
				t.Errorf("err = %v (kind %v, role %q), want kind %v role %q", r.Err, errs.KindOf(r.Err), errs.RoleOf(r.Err), tt.wantKind, tt.wantRole)
			}
			if hasVia := contains(r.Err.Error(), "via bastion jump"); hasVia != tt.wantVia {
				t.Errorf("message %q: via mention = %v, want %v", r.Err, hasVia, tt.wantVia)
			}
			if len(target.Commands()) != 0 {
				t.Error("the target ran a command")
			}
		})
	}
}
