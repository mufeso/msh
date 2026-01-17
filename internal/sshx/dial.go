package sshx

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/errs"
)

// Dialer opens SSH connections.
type Dialer struct {
	cfg      Config
	auth     *Auth
	hostKeys ssh.HostKeyCallback
}

// NewDialer prepares a Dialer. It fails if host key verification cannot
// be set up (for example a missing known_hosts file).
func NewDialer(cfg Config, auth *Auth) (*Dialer, error) {
	cb, err := HostKeyCallback(cfg)
	if err != nil {
		return nil, errs.New(errs.KindHostKey, "", err)
	}
	return &Dialer{cfg: cfg, auth: auth, hostKeys: cb}, nil
}

// Dial connects to ep: TCP connect, then SSH handshake and
// authentication, all bounded by cfg.ConnectTimeout and ctx.
func (d *Dialer) Dial(ctx context.Context, ep Endpoint) (*ssh.Client, error) {
	nd := net.Dialer{Timeout: d.cfg.ConnectTimeout}
	conn, err := nd.DialContext(ctx, "tcp", ep.Addr())
	if err != nil {
		return nil, classifyDial(ctx, ep, err)
	}
	return d.Handshake(ctx, conn, ep)
}

// Handshake runs the SSH handshake over an already established
// connection. It takes ownership of conn: on failure conn is closed.
// Keeping it separate from Dial lets bastion connections reuse it.
func (d *Dialer) Handshake(ctx context.Context, conn net.Conn, ep Endpoint) (*ssh.Client, error) {
	if t := d.cfg.ConnectTimeout; t > 0 {
		_ = conn.SetDeadline(time.Now().Add(t))
	}
	// If ctx ends during the handshake, closing the connection makes the
	// blocked handshake return immediately. stop() unregisters the
	// callback; it returns false if the callback already ran.
	stop := context.AfterFunc(ctx, func() { conn.Close() })

	c, chans, reqs, err := ssh.NewClientConn(conn, ep.Addr(), ClientConfig(d.cfg, ep, d.auth, d.hostKeys))
	interrupted := !stop()
	if err != nil || interrupted {
		conn.Close()
		if c != nil {
			c.Close()
		}
		if interrupted {
			return nil, contextError(ctx, ep)
		}
		return nil, classifyHandshake(ep, err)
	}
	_ = conn.SetDeadline(time.Time{}) // no deadline once connected
	return ssh.NewClient(c, chans, reqs), nil
}

// classifyDial categorizes a TCP connection error.
func classifyDial(ctx context.Context, ep Endpoint, err error) error {
	if ctx.Err() != nil {
		return contextError(ctx, ep)
	}
	var dnsErr *net.DNSError
	var netErr net.Error
	switch {
	case errors.As(err, &dnsErr):
		return errs.New(errs.KindDNS, ep.Name, err)
	case errors.As(err, &netErr) && netErr.Timeout():
		return errs.New(errs.KindConnectTimeout, ep.Name, err)
	default:
		return errs.New(errs.KindTCP, ep.Name, err)
	}
}

// classifyHandshake categorizes an SSH handshake error.
//
// x/crypto/ssh reports authentication failure only as text, so this is
// the one place where msh inspects an error string.
func classifyHandshake(ep Endpoint, err error) error {
	var hkErr *HostKeyError
	var netErr net.Error
	switch {
	case errors.As(err, &hkErr):
		return errs.New(errs.KindHostKey, ep.Name, hkErr)
	case strings.Contains(err.Error(), "unable to authenticate"):
		return errs.New(errs.KindAuth, ep.Name, err)
	case errors.As(err, &netErr) && netErr.Timeout():
		return errs.New(errs.KindConnectTimeout, ep.Name, err)
	default:
		return errs.New(errs.KindHandshake, ep.Name, err)
	}
}

// contextError reports why ctx ended.
func contextError(ctx context.Context, ep Endpoint) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errs.New(errs.KindConnectTimeout, ep.Name, ctx.Err())
	}
	return errs.New(errs.KindCanceled, ep.Name, ctx.Err())
}
