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

// creds are the credentials and checks used for one hop: client
// configuration, keys, and host key verification.
type creds struct {
	cfg      Config
	auth     *Auth
	hostKeys ssh.HostKeyCallback
	ownsAuth bool // auth was created for this hop and must be closed with it
}

// bastionHop is everything needed to connect to one bastion.
type bastionHop struct {
	ep    Endpoint
	creds creds
}

// BastionSpec describes a bastion for AddBastion.
type BastionSpec struct {
	Name           string
	Endpoint       Endpoint
	IdentityFile   string // "" = the same keys as for target servers
	KnownHostsFile string // "" = the same known_hosts as for target servers
}

// Dialer opens SSH connections, directly or through bastions.
type Dialer struct {
	target   creds
	bastions map[string]*bastionHop
}

// NewDialer prepares a Dialer. It fails if host key verification cannot
// be set up (for example a missing known_hosts file).
func NewDialer(cfg Config, auth *Auth) (*Dialer, error) {
	cb, err := HostKeyCallback(cfg)
	if err != nil {
		return nil, errs.New(errs.KindHostKey, "", err)
	}
	return &Dialer{
		target:   creds{cfg: cfg, auth: auth, hostKeys: cb},
		bastions: map[string]*bastionHop{},
	}, nil
}

// AddBastion registers a bastion. A bastion-specific identity file gets
// its own Auth; a bastion-specific known_hosts file its own callback.
func (d *Dialer) AddBastion(spec BastionSpec) error {
	c := d.target // a copy: changing c does not change d.target
	c.ownsAuth = false
	if spec.KnownHostsFile != "" {
		c.cfg.KnownHostsFile = spec.KnownHostsFile
		cb, err := HostKeyCallback(c.cfg)
		if err != nil {
			return errs.NewRole(errs.KindHostKey, errs.RoleBastion, spec.Name, err)
		}
		c.hostKeys = cb
	}
	if spec.IdentityFile != "" {
		c.cfg.IdentityFile = spec.IdentityFile
		auth, err := NewAuth(c.cfg)
		if err != nil {
			return errs.NewRole(errs.KindAuth, errs.RoleBastion, spec.Name, err)
		}
		c.auth, c.ownsAuth = auth, true
	}
	d.bastions[spec.Name] = &bastionHop{ep: spec.Endpoint, creds: c}
	return nil
}

// Close releases resources the Dialer created for bastions.
func (d *Dialer) Close() error {
	for _, b := range d.bastions {
		if b.creds.ownsAuth {
			b.creds.auth.Close()
		}
	}
	return nil
}

// Dial connects directly to a target endpoint.
func (d *Dialer) Dial(ctx context.Context, ep Endpoint) (*ssh.Client, error) {
	return d.dial(ctx, ep, d.target)
}

// Handshake runs the target SSH handshake over an established connection
// (for example one forwarded by a bastion). It takes ownership of conn.
func (d *Dialer) Handshake(ctx context.Context, conn net.Conn, ep Endpoint) (*ssh.Client, error) {
	return d.handshake(ctx, conn, ep, d.target)
}

// DialBastion connects to the named bastion. Errors carry the bastion role.
func (d *Dialer) DialBastion(ctx context.Context, name string) (*ssh.Client, error) {
	hop, ok := d.bastions[name]
	if !ok {
		return nil, errs.NewRole(errs.KindConfig, errs.RoleBastion, name, errors.New("unknown bastion"))
	}
	c, err := d.dial(ctx, hop.ep, hop.creds)
	if err != nil {
		return nil, asBastionError(err)
	}
	return c, nil
}

// dial makes the TCP connection and then the SSH handshake.
func (d *Dialer) dial(ctx context.Context, ep Endpoint, c creds) (*ssh.Client, error) {
	nd := net.Dialer{Timeout: c.cfg.ConnectTimeout}
	conn, err := nd.DialContext(ctx, "tcp", ep.Addr())
	if err != nil {
		return nil, classifyDial(ctx, ep, err)
	}
	return d.handshake(ctx, conn, ep, c)
}

// handshake runs the SSH handshake and authentication over conn.
func (d *Dialer) handshake(ctx context.Context, conn net.Conn, ep Endpoint, c creds) (*ssh.Client, error) {
	if t := c.cfg.ConnectTimeout; t > 0 {
		// Forwarded connections do not support deadlines; the error is
		// ignored because ctx still bounds the handshake.
		_ = conn.SetDeadline(time.Now().Add(t))
	}
	// If ctx ends during the handshake, closing the connection makes the
	// blocked handshake return immediately. stop() returns false if the
	// callback already ran.
	stop := context.AfterFunc(ctx, func() { conn.Close() })

	cfg := c.cfg
	if cfg.Password != nil {
		cfg.Password = pauseTimeouts(ctx, conn, cfg)
	}
	sc, chans, reqs, err := ssh.NewClientConn(conn, ep.Addr(), ClientConfig(cfg, ep, c.auth, c.hostKeys))
	interrupted := !stop()
	if err != nil || interrupted {
		conn.Close()
		if sc != nil {
			sc.Close()
		}
		if interrupted {
			return nil, contextError(ctx, ep)
		}
		return nil, classifyHandshake(ep, err)
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(sc, chans, reqs), nil
}

// pauseTimeouts stops the connect timeouts while the password is fetched:
// a person typing should not count as a slow server.
func pauseTimeouts(ctx context.Context, conn net.Conn, cfg Config) PasswordFunc {
	return func(user, host string) (string, error) {
		defer pauseClock(ctx)()
		_ = conn.SetDeadline(time.Time{})
		if t := cfg.ConnectTimeout; t > 0 {
			defer func() { _ = conn.SetDeadline(time.Now().Add(t)) }()
		}
		return cfg.Password(user, host)
	}
}

// asBastionError marks a connection error as a bastion failure.
func asBastionError(err error) error {
	var e *errs.Error
	if errors.As(err, &e) {
		copied := *e
		copied.Role = errs.RoleBastion
		return &copied
	}
	return errs.NewRole(errs.KindHandshake, errs.RoleBastion, "", err)
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
	if cause := context.Cause(ctx); errors.Is(cause, context.DeadlineExceeded) {
		return errs.New(errs.KindConnectTimeout, ep.Name, cause)
	}
	return errs.New(errs.KindCanceled, ep.Name, ctx.Err())
}
