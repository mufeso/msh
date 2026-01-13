package sshx

import (
	"testing"
	"time"

	"example.com/msh/internal/inventory"
)

func TestEndpointFor(t *testing.T) {
	ep := EndpointFor(inventory.Server{Name: "web01", Address: "2001:db8::10", Port: 2222, User: "admin"})
	if ep.Addr() != "[2001:db8::10]:2222" {
		t.Errorf("Addr() = %q", ep.Addr())
	}
	if ep.String() != "admin@[2001:db8::10]:2222" {
		t.Errorf("String() = %q", ep.String())
	}
}

func TestBaseClientConfig(t *testing.T) {
	cc := baseClientConfig(Config{ConnectTimeout: 3 * time.Second}, Endpoint{User: "ops"})
	if cc.User != "ops" || cc.ClientVersion != "SSH-2.0-msh" || cc.Timeout != 3*time.Second {
		t.Errorf("config = %+v", cc)
	}
}
