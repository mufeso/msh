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
	d.AddBastion("jump", endpointOf("jump", env.server))
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
	d.AddBastion("jump", endpointOf("jump", env.server))
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
