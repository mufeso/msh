package sshx

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/sshtest"
)

// bastionPair starts a target and a bastion that both accept env's key.
func bastionPair(t *testing.T) (*testEnv, *sshtest.Server) {
	t.Helper()
	env := newTestEnv(t, sshtest.Options{})
	bastion := sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{env.key.Public()}, AllowForwarding: true})
	return env, bastion
}

func TestBastionOwnKnownHostsFile(t *testing.T) {
	env, bastion := bastionPair(t)
	// The global known_hosts only knows the target...
	env.cfg.KnownHostsFile = sshtest.WriteKnownHosts(t, t.TempDir(), env.server.KnownHostsLine())
	d := env.dialer(t)
	defer d.Close()
	// ...and the bastion's own file knows the bastion.
	mustAddBastion(t, d, BastionSpec{
		Name:           "jump",
		Endpoint:       endpointOf("jump", bastion),
		KnownHostsFile: sshtest.WriteKnownHosts(t, t.TempDir(), bastion.KnownHostsLine()),
	})
	target := env.server0()
	target.Bastion = "jump"
	if r := (&Executor{Connector: d}).Run(context.Background(), target, executor.Request{Command: "true"}); r.Status != executor.StatusOK {
		t.Fatalf("result = %+v", r)
	}
}

func TestBastionChangedHostKeyStopsEverything(t *testing.T) {
	env, bastion := bastionPair(t)
	// The bastion's recorded key is a different key: a possible attack.
	forged := sshtest.KnownHostsLine(bastion.Addr, sshtest.NewKey(t).Public())
	env.cfg.KnownHostsFile = sshtest.WriteKnownHosts(t, t.TempDir(), env.server.KnownHostsLine(), forged)
	d := env.dialer(t)
	defer d.Close()
	mustAddBastion(t, d, BastionSpec{Name: "jump", Endpoint: endpointOf("jump", bastion)})

	target := env.server0()
	target.Bastion = "jump"
	r := (&Executor{Connector: d}).Run(context.Background(), target, executor.Request{Command: "true"})

	var hk *HostKeyError
	if errs.KindOf(r.Err) != errs.KindHostKey || errs.RoleOf(r.Err) != errs.RoleBastion || !errors.As(r.Err, &hk) || hk.Unknown {
		t.Fatalf("err = %v, want bastion host key mismatch", r.Err)
	}
	if fw := bastion.Forwards(); len(fw) != 0 {
		t.Errorf("forwarding happened after a failed host key check: %v", fw)
	}
	if cmds := env.server.Commands(); len(cmds) != 0 {
		t.Errorf("target was contacted (no fallback allowed): %v", cmds)
	}
}

func TestBastionMissingKnownHostsFile(t *testing.T) {
	env, bastion := bastionPair(t)
	d := env.dialer(t)
	err := d.AddBastion(BastionSpec{Name: "jump", Endpoint: endpointOf("jump", bastion), KnownHostsFile: t.TempDir() + "/missing"})
	if errs.KindOf(err) != errs.KindHostKey || errs.RoleOf(err) != errs.RoleBastion {
		t.Errorf("err = %v", err)
	}
}
