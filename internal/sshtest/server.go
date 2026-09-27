package sshtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Exec describes one command the test server was asked to run.
type Exec struct {
	Command string // "" for an interactive shell request
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	PTY     *PTY // non-nil when the client requested a pseudo-terminal
}

// PTY describes a pseudo-terminal request.
type PTY struct {
	Term          string
	Width, Height int
}

// Handler runs a command and returns its exit status. ctx is canceled
// when the client sends a signal or closes the session.
type Handler func(ctx context.Context, e *Exec) int

// Options configure a Server.
type Options struct {
	AuthorizedKeys  []ssh.PublicKey // keys allowed to log in (any user name)
	Handler         Handler         // default: ShellHandler
	AllowForwarding bool            // accept direct-tcpip channels (act as a bastion)
	Password        string          // also accept this password ("" = keys only)
	KeyboardOnly    bool            // ask for Password with keyboard-interactive only
}

// Server is an SSH server listening on 127.0.0.1 for one test.
type Server struct {
	Addr    string // host:port to connect to
	HostKey Key    // the server's host key

	passwordTries atomic.Int32
	agentKeys     atomic.Int32

	opts Options
	cfg  *ssh.ServerConfig
	ln   net.Listener
	wg   sync.WaitGroup

	mu       sync.Mutex
	conns    map[net.Conn]struct{}
	commands []string
	signals  []string
	forwards []string
	remotes  []string
	ptys     []PTY
	resizes  [][2]int // width, height

	active atomic.Int64 // sessions currently open
}

// NewServer starts a server; it is closed automatically when the test ends.
func NewServer(t testing.TB, opts Options) *Server {
	t.Helper()
	s := &Server{HostKey: NewKey(t), opts: opts, conns: map[net.Conn]struct{}{}}
	if s.opts.Handler == nil {
		s.opts.Handler = ShellHandler
	}
	s.cfg = &ssh.ServerConfig{
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			for _, k := range s.opts.AuthorizedKeys {
				if bytes.Equal(k.Marshal(), key.Marshal()) {
					return &ssh.Permissions{}, nil
				}
			}
			return nil, fmt.Errorf("key not authorized for %q", meta.User())
		},
	}
	if s.opts.KeyboardOnly {
		s.cfg.KeyboardInteractiveCallback = func(meta ssh.ConnMetadata, challenge ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
			answers, err := challenge("", "", []string{"Password: "}, []bool{false})
			if err != nil {
				return nil, err
			}
			return s.checkPassword(meta, answers[0])
		}
	} else if s.opts.Password != "" {
		s.cfg.PasswordCallback = func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			return s.checkPassword(meta, string(password))
		}
	}
	s.cfg.AddHostKey(s.HostKey.Signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0") // port 0: pick a free port
	if err != nil {
		t.Fatal(err)
	}
	s.ln, s.Addr = ln, ln.Addr().String()

	s.wg.Add(1)
	go s.serve()
	t.Cleanup(s.Close)
	return s
}

// checkPassword counts the attempt and compares the password.
func (s *Server) checkPassword(meta ssh.ConnMetadata, password string) (*ssh.Permissions, error) {
	s.passwordTries.Add(1)
	if password == s.opts.Password {
		return &ssh.Permissions{}, nil
	}
	return nil, fmt.Errorf("wrong password for %q", meta.User())
}

// countAgentKeys does what a program on the server would do with a
// forwarded agent: it opens a channel back to the client and lists the keys.
func (s *Server) countAgentKeys(conn *ssh.ServerConn) {
	ch, reqs, err := conn.OpenChannel("auth-agent@openssh.com", nil)
	if err != nil {
		return
	}
	defer ch.Close()
	go ssh.DiscardRequests(reqs)
	if keys, err := agent.NewClient(ch).List(); err == nil {
		s.agentKeys.Store(int32(len(keys)))
	}
}

// AgentKeys returns how many keys the server saw through a forwarded agent.
func (s *Server) AgentKeys() int { return int(s.agentKeys.Load()) }

// PasswordTries returns how many passwords the server has received.
func (s *Server) PasswordTries() int { return int(s.passwordTries.Load()) }

// KnownHostsLine returns the known_hosts line for this server.
func (s *Server) KnownHostsLine() string {
	return KnownHostsLine(s.Addr, s.HostKey.Public())
}

// Commands returns the commands received so far.
func (s *Server) Commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

// Signals returns the signal names received so far (e.g. "TERM").
func (s *Server) Signals() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.signals...)
}

// PTYs returns the pseudo-terminal requests received so far.
func (s *Server) PTYs() []PTY {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]PTY(nil), s.ptys...)
}

// Resizes returns the window-change requests as [width, height] pairs.
func (s *Server) Resizes() [][2]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][2]int(nil), s.resizes...)
}

// RemoteForwards returns the addresses clients asked the server to listen on.
func (s *Server) RemoteForwards() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.remotes)
}

// handleGlobal answers requests that are not about one channel. The only
// ones the server knows are for remote forwarding ("ssh -R").
func (s *Server) handleGlobal(conn *ssh.ServerConn, reqs <-chan *ssh.Request) {
	listeners := map[string]net.Listener{}
	defer func() {
		for _, ln := range listeners {
			ln.Close()
		}
	}()
	for req := range reqs {
		var p struct {
			Addr string
			Port uint32
		}
		switch {
		case req.Type == "tcpip-forward" && s.opts.AllowForwarding && ssh.Unmarshal(req.Payload, &p) == nil:
			ln, err := net.Listen("tcp", net.JoinHostPort(p.Addr, strconv.Itoa(int(p.Port))))
			if err != nil {
				req.Reply(false, nil)
				continue
			}
			port := uint32(ln.Addr().(*net.TCPAddr).Port)
			addr := net.JoinHostPort(p.Addr, strconv.Itoa(int(port)))
			listeners[addr] = ln
			s.record(&s.remotes, addr)
			req.Reply(true, ssh.Marshal(struct{ Port uint32 }{port}))
			go s.serveRemoteForward(conn, ln, p.Addr, port)
		case req.Type == "cancel-tcpip-forward" && ssh.Unmarshal(req.Payload, &p) == nil:
			addr := net.JoinHostPort(p.Addr, strconv.Itoa(int(p.Port)))
			if ln, ok := listeners[addr]; ok {
				ln.Close()
				delete(listeners, addr)
			}
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

// serveRemoteForward opens a channel to the client for every connection.
func (s *Server) serveRemoteForward(conn *ssh.ServerConn, ln net.Listener, addr string, port uint32) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer c.Close()
			from := c.RemoteAddr().(*net.TCPAddr)
			ch, reqs, err := conn.OpenChannel("forwarded-tcpip", ssh.Marshal(struct {
				Addr     string
				Port     uint32
				OrigAddr string
				OrigPort uint32
			}{addr, port, from.IP.String(), uint32(from.Port)}))
			if err != nil {
				return
			}
			defer ch.Close()
			go ssh.DiscardRequests(reqs)
			done := make(chan struct{})
			go func() {
				defer close(done)
				io.Copy(ch, c)
				ch.CloseWrite()
			}()
			io.Copy(c, ch)
			c.(*net.TCPConn).CloseWrite()
			<-done
		}()
	}
}

// Forwards returns the host:port targets of direct-tcpip channels.
func (s *Server) Forwards() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.forwards...)
}

// ActiveSessions returns the number of open sessions. Tests use it to
// prove that clients clean up.
func (s *Server) ActiveSessions() int64 { return s.active.Load() }

// WaitIdle waits until no session is open or the timeout expires.
func (s *Server) WaitIdle(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.ActiveSessions() == 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return s.ActiveSessions() == 0
}

// Close stops the server and waits for all its goroutines.
func (s *Server) Close() {
	s.ln.Close()
	s.mu.Lock()
	for c := range s.conns {
		c.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Server) serve() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return // listener closed
		}
		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() {
				conn.Close()
				s.mu.Lock()
				delete(s.conns, conn)
				s.mu.Unlock()
			}()
			s.handleConn(conn)
		}()
	}
}

func (s *Server) handleConn(nc net.Conn) {
	sconn, chans, reqs, err := ssh.NewServerConn(nc, s.cfg)
	if err != nil {
		return // handshake or authentication failed
	}
	defer sconn.Close()
	go s.handleGlobal(sconn, reqs)

	var wg sync.WaitGroup
	for newCh := range chans {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handleChannel(sconn, newCh)
		}()
	}
	wg.Wait()
}

func (s *Server) handleChannel(conn *ssh.ServerConn, newCh ssh.NewChannel) {
	switch newCh.ChannelType() {
	case "session":
		s.handleSession(conn, newCh)
	case "direct-tcpip":
		s.handleDirectTCPIP(newCh)
	default:
		newCh.Reject(ssh.UnknownChannelType, "unsupported channel type")
	}
}

func (s *Server) handleSession(conn *ssh.ServerConn, newCh ssh.NewChannel) {
	ch, reqs, err := newCh.Accept()
	if err != nil {
		return
	}
	s.active.Add(1)
	defer s.active.Add(-1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var done chan struct{}
	var pty *PTY

	start := func(command string) {
		done = make(chan struct{})
		go func() {
			defer close(done)
			s.run(ctx, ch, command, pty)
		}()
	}

	for req := range reqs {
		switch req.Type {
		case "exec":
			var p struct{ Command string }
			if ssh.Unmarshal(req.Payload, &p) != nil || done != nil {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			s.record(&s.commands, p.Command)
			start(p.Command)
		case "shell":
			if done != nil {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			s.record(&s.commands, "")
			start("")
		case "pty-req":
			var p struct {
				Term                         string
				Columns, Rows, Width, Height uint32
				Modes                        string
			}
			if ssh.Unmarshal(req.Payload, &p) != nil {
				req.Reply(false, nil)
				continue
			}
			pty = &PTY{Term: p.Term, Width: int(p.Columns), Height: int(p.Rows)}
			s.mu.Lock()
			s.ptys = append(s.ptys, *pty)
			s.mu.Unlock()
			req.Reply(true, nil)
		case "window-change":
			var p struct{ Columns, Rows, Width, Height uint32 }
			if ssh.Unmarshal(req.Payload, &p) == nil {
				s.mu.Lock()
				s.resizes = append(s.resizes, [2]int{int(p.Columns), int(p.Rows)})
				s.mu.Unlock()
			}
		case "signal":
			var p struct{ Signal string }
			_ = ssh.Unmarshal(req.Payload, &p)
			s.record(&s.signals, p.Signal)
			cancel()
		case "auth-agent-req@openssh.com":
			req.Reply(true, nil)
			s.countAgentKeys(conn)
		case "env":
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
	// The client closed the channel: stop the command and wait for it.
	cancel()
	if done != nil {
		<-done
	}
	ch.Close()
}

func (s *Server) run(ctx context.Context, ch ssh.Channel, command string, pty *PTY) {
	code := s.opts.Handler(ctx, &Exec{Command: command, Stdin: ch, Stdout: ch, Stderr: ch.Stderr(), PTY: pty})
	status := struct{ Status uint32 }{uint32(code)}
	_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(&status))
	ch.Close()
}

func (s *Server) record(list *[]string, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	*list = append(*list, v)
}

// ShellHandler runs the command on the local machine with /bin/sh -c,
// which makes the test server behave like a real one.
func ShellHandler(ctx context.Context, e *Exec) int {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", e.Command)
	if e.Command == "" {
		// A shell request: run a shell that reads commands from stdin.
		cmd = exec.CommandContext(ctx, "/bin/sh")
	}
	cmd.Stdout, cmd.Stderr = e.Stdout, e.Stderr
	cmd.WaitDelay = time.Second
	// Like sshd, the command's end must not depend on the client closing
	// stdin: copy stdin in a goroutine that Wait does not wait for.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return 255
	}
	if err := cmd.Start(); err != nil {
		return 255
	}
	go func() {
		_, _ = io.Copy(stdin, e.Stdin)
		stdin.Close()
	}()
	err = cmd.Wait()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exitErr) && exitErr.ExitCode() >= 0:
		return exitErr.ExitCode()
	default:
		return 255
	}
}

// Static returns a handler that always prints stdout and stderr and exits
// with code.
func Static(stdout, stderr string, code int) Handler {
	return func(_ context.Context, e *Exec) int {
		io.WriteString(e.Stdout, stdout)
		io.WriteString(e.Stderr, stderr)
		return code
	}
}

// Blocking returns a handler that prints started and then waits until
// the session is signaled or closed.
func Blocking(started string) Handler {
	return func(ctx context.Context, e *Exec) int {
		io.WriteString(e.Stdout, started)
		<-ctx.Done()
		return 143
	}
}

// handleDirectTCPIP forwards a TCP connection, like "ssh -J" asks a
// bastion to do. The payload names the target host and port.
func (s *Server) handleDirectTCPIP(newCh ssh.NewChannel) {
	if !s.opts.AllowForwarding {
		newCh.Reject(ssh.Prohibited, "port forwarding is disabled")
		return
	}
	var p struct {
		Host     string
		Port     uint32
		OrigHost string
		OrigPort uint32
	}
	if err := ssh.Unmarshal(newCh.ExtraData(), &p); err != nil {
		newCh.Reject(ssh.ConnectionFailed, "bad payload")
		return
	}
	target := net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port)))
	s.record(&s.forwards, target)

	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		newCh.Reject(ssh.ConnectionFailed, err.Error())
		return
	}
	defer conn.Close()
	ch, reqs, err := newCh.Accept()
	if err != nil {
		return
	}
	defer ch.Close()
	go ssh.DiscardRequests(reqs)

	// Copy in both directions. When one direction ends, signal EOF to the
	// other side with CloseWrite and wait for the second copy.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(conn, ch)
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		io.Copy(ch, conn)
		ch.CloseWrite()
	}()
	wg.Wait()
}
