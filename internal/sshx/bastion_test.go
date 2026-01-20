package sshx

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/sshtest"
	"golang.org/x/crypto/ssh"
)

// endpointOf returns an Endpoint for an in-process server.
func endpointOf(name string, srv *sshtest.Server) Endpoint {
	host, portStr, _ := net.SplitHostPort(srv.Addr)
	port, _ := strconv.Atoi(portStr)
	return Endpoint{Name: name, Host: host, Port: port, User: "tester"}
}

func TestDialBastion(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{AllowForwarding: true})
	d := env.dialer(t)
	d.AddBastion("jump", endpointOf("jump", env.server))

	c, err := d.DialBastion(context.Background(), "jump")
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}

func TestDialBastionErrorsHaveBastionRole(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{sshtest.NewKey(t).Public()}})
	d := env.dialer(t)
	d.AddBastion("jump", endpointOf("jump", env.server))

	_, err := d.DialBastion(context.Background(), "jump")
	if errs.KindOf(err) != errs.KindAuth || errs.RoleOf(err) != errs.RoleBastion {
		t.Errorf("err = %v, want bastion authentication failure", err)
	}
	if !contains(err.Error(), "bastion jump: authentication failed") {
		t.Errorf("message = %q", err)
	}

	if _, err := d.DialBastion(context.Background(), "ghost"); errs.RoleOf(err) != errs.RoleBastion {
		t.Errorf("unknown bastion: %v", err)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
