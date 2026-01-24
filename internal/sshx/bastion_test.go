package sshx

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.com/msh/internal/errs"
	"example.com/msh/internal/executor"
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
	mustAddBastion(t, d, BastionSpec{Name: "jump", Endpoint: endpointOf("jump", env.server)})

	c, err := d.DialBastion(context.Background(), "jump")
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
}

func TestDialBastionErrorsHaveBastionRole(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{sshtest.NewKey(t).Public()}})
	d := env.dialer(t)
	mustAddBastion(t, d, BastionSpec{Name: "jump", Endpoint: endpointOf("jump", env.server)})

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

// startEcho starts a TCP server that echoes one line.
func startEcho(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				io.Copy(c, c)
			}()
		}
	}()
	return ln.Addr().String()
}

func TestForwardThroughBastion(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{AllowForwarding: true})
	d := env.dialer(t)
	mustAddBastion(t, d, BastionSpec{Name: "jump", Endpoint: endpointOf("jump", env.server)})
	bastion, err := d.DialBastion(context.Background(), "jump")
	if err != nil {
		t.Fatal(err)
	}
	defer bastion.Close()

	echoAddr := startEcho(t)
	host, portStr, _ := net.SplitHostPort(echoAddr)
	port, _ := strconv.Atoi(portStr)
	conn, err := Forward(context.Background(), bastion, "jump", Endpoint{Name: "echo", Host: host, Port: port})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintln(conn, "ping")
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil || line != "ping\n" {
		t.Errorf("echo = %q, %v", line, err)
	}
	if fw := env.server.Forwards(); len(fw) != 1 || fw[0] != echoAddr {
		t.Errorf("bastion forwards = %v", fw)
	}
}

func TestForwardRefusedIsTargetTCPError(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{AllowForwarding: true})
	d := env.dialer(t)
	mustAddBastion(t, d, BastionSpec{Name: "jump", Endpoint: endpointOf("jump", env.server)})
	bastion, err := d.DialBastion(context.Background(), "jump")
	if err != nil {
		t.Fatal(err)
	}
	defer bastion.Close()

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	_, err = Forward(context.Background(), bastion, "jump", Endpoint{Name: "app01", Host: "127.0.0.1", Port: port})
	if errs.KindOf(err) != errs.KindTCP || errs.RoleOf(err) != errs.RoleTarget || !contains(err.Error(), "via bastion jump") {
		t.Errorf("err = %v", err)
	}
}

func TestConnectThroughBastion(t *testing.T) {
	// One key is authorized on both servers; the target only listens on
	// 127.0.0.1, and the bastion forwards to it.
	env := newTestEnv(t, sshtest.Options{})
	bastion := sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{env.key.Public()}, AllowForwarding: true})
	env.cfg.KnownHostsFile = sshtest.WriteKnownHosts(t, t.TempDir(), env.server.KnownHostsLine(), bastion.KnownHostsLine())
	d := env.dialer(t)
	mustAddBastion(t, d, BastionSpec{Name: "jump", Endpoint: endpointOf("jump", bastion)})

	target := env.server0()
	target.Bastion = "jump"
	ex := &Executor{Connector: d}
	r := ex.Run(context.Background(), target, executor.Request{Command: "echo through-bastion"})
	if r.Status != executor.StatusOK || r.Stdout != "through-bastion\n" {
		t.Fatalf("result = %+v", r)
	}
	if fw := bastion.Forwards(); len(fw) != 1 || fw[0] != env.server.Addr {
		t.Errorf("bastion forwards = %v, want [%s]", fw, env.server.Addr)
	}
	if !env.server.WaitIdle(2 * time.Second) {
		t.Error("target session left open")
	}
}

func TestBastionFailureDoesNotFallBackToDirect(t *testing.T) {
	env := newTestEnv(t, sshtest.Options{})
	d := env.dialer(t)
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	mustAddBastion(t, d, BastionSpec{Name: "jump", Endpoint: Endpoint{Name: "jump", Host: "127.0.0.1", Port: port, User: "u"}})

	target := env.server0() // directly reachable, but configured via the bastion
	target.Bastion = "jump"
	r := (&Executor{Connector: d}).Run(context.Background(), target, executor.Request{Command: "true"})
	if r.Status != executor.StatusError || errs.RoleOf(r.Err) != errs.RoleBastion || errs.KindOf(r.Err) != errs.KindTCP {
		t.Errorf("result = %+v", r)
	}
	if len(env.server.Commands()) != 0 {
		t.Error("msh fell back to a direct connection")
	}
}

func mustAddBastion(t *testing.T, d *Dialer, spec BastionSpec) {
	t.Helper()
	if err := d.AddBastion(spec); err != nil {
		t.Fatal(err)
	}
}

func TestBastionOwnIdentity(t *testing.T) {
	// The target accepts the normal key; the bastion only a second key
	// that is configured as the bastion's identity_file.
	env := newTestEnv(t, sshtest.Options{})
	jumpKey := sshtest.NewKey(t)
	bastion := sshtest.NewServer(t, sshtest.Options{AuthorizedKeys: []ssh.PublicKey{jumpKey.Public()}, AllowForwarding: true})
	env.cfg.KnownHostsFile = sshtest.WriteKnownHosts(t, t.TempDir(), env.server.KnownHostsLine(), bastion.KnownHostsLine())
	d := env.dialer(t)
	defer d.Close()
	mustAddBastion(t, d, BastionSpec{
		Name:         "jump",
		Endpoint:     endpointOf("jump", bastion),
		IdentityFile: sshtest.WriteKeyFile(t, t.TempDir(), "jump_key", jumpKey, ""),
	})
	target := env.server0()
	target.Bastion = "jump"
	r := (&Executor{Connector: d}).Run(context.Background(), target, executor.Request{Command: "echo ok"})
	if r.Status != executor.StatusOK {
		t.Fatalf("result = %+v", r)
	}
}
