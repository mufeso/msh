package sshx

import (
	"io"
	"testing"

	"golang.org/x/crypto/ssh"
)

// recordingSession shows how small a test double for Session can be.
type recordingSession struct {
	started string
}

func (r *recordingSession) SetStdio(io.Reader, io.Writer, io.Writer) {}
func (r *recordingSession) Start(cmd string) error                   { r.started = cmd; return nil }
func (r *recordingSession) Wait() error                              { return nil }
func (r *recordingSession) Signal(ssh.Signal) error                  { return nil }
func (r *recordingSession) Close() error                             { return nil }
func (r *recordingSession) RequestPty(string, int, int, ssh.TerminalModes) error {
	return nil
}
func (r *recordingSession) Shell() error                { return nil }
func (r *recordingSession) WindowChange(int, int) error { return nil }

func TestSessionInterfaceIsFakeable(t *testing.T) {
	var s Session = &recordingSession{}
	if err := s.Start("uptime"); err != nil {
		t.Fatal(err)
	}
	if got := s.(*recordingSession).started; got != "uptime" {
		t.Errorf("started = %q", got)
	}
}
