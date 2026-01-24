// Package errs defines the error categories of msh.
//
// Code that detects a failure wraps it in an *Error with a Kind. Code that
// reports failures (output, exit codes) inspects the Kind with KindOf and
// never parses error strings.
package errs

import (
	"context"
	"errors"
	"strings"

	"example.com/msh/internal/exitcode"
)

// Kind is a failure category.
type Kind int

// Failure categories.
const (
	KindUnknown Kind = iota
	KindUsage
	KindConfig
	KindInventory
	KindTask
	KindTaskArgs
	KindDNS
	KindTCP
	KindHandshake
	KindAuth
	KindHostKey
	KindConnectTimeout
	KindRemoteCommand
	KindRemoteTimeout
	KindScriptTransfer
	KindBinaryTransfer
	KindCanceled
	KindGlobalTimeout
	KindTmuxUnavailable
	KindTmuxStartup
	KindFileRead
	KindFileWrite
	KindInternal
)

var kindText = map[Kind]string{
	KindUnknown:         "error",
	KindUsage:           "invalid command-line input",
	KindConfig:          "invalid configuration",
	KindInventory:       "invalid inventory",
	KindTask:            "invalid task",
	KindTaskArgs:        "invalid task arguments",
	KindDNS:             "DNS lookup failed",
	KindTCP:             "TCP connection failed",
	KindHandshake:       "SSH handshake failed",
	KindAuth:            "authentication failed",
	KindHostKey:         "host key verification failed",
	KindConnectTimeout:  "connection timed out",
	KindRemoteCommand:   "remote command failed",
	KindRemoteTimeout:   "remote command timed out",
	KindScriptTransfer:  "script transfer failed",
	KindBinaryTransfer:  "binary transfer failed",
	KindCanceled:        "canceled",
	KindGlobalTimeout:   "global timeout expired",
	KindTmuxUnavailable: "tmux is not available",
	KindTmuxStartup:     "tmux failed",
	KindFileRead:        "cannot read file",
	KindFileWrite:       "cannot write file",
	KindInternal:        "internal error",
}

// String returns a short human-readable description.
func (k Kind) String() string {
	if s, ok := kindText[k]; ok {
		return s
	}
	return "error"
}

// Role says which hop of a connection failed.
type Role string

// Connection roles.
const (
	RoleTarget  Role = ""        // the server itself (the default)
	RoleBastion Role = "bastion" // a bastion (jump host) on the way
)

// Error is a categorized error.
type Error struct {
	Kind Kind
	Role Role   // which hop failed, for connection errors
	Host string // server or bastion name, if the error concerns one
	Via  string // bastion the target was reached through, if any
	Err  error  // underlying cause; may be nil
}

// NewRole returns an *Error for a specific connection hop.
func NewRole(kind Kind, role Role, host string, err error) *Error {
	return &Error{Kind: kind, Role: role, Host: host, Err: err}
}

// RoleOf returns the Role of the first *Error in err's chain.
func RoleOf(err error) Role {
	var e *Error
	if errors.As(err, &e) {
		return e.Role
	}
	return RoleTarget
}

// New returns an *Error.
func New(kind Kind, host string, err error) *Error {
	return &Error{Kind: kind, Host: host, Err: err}
}

// Error formats "host: kind: cause".
func (e *Error) Error() string {
	var b strings.Builder
	if e.Role != RoleTarget {
		b.WriteString(string(e.Role))
		b.WriteString(" ")
	}
	if e.Host != "" {
		b.WriteString(e.Host)
		if e.Via != "" {
			b.WriteString(" (via bastion ")
			b.WriteString(e.Via)
			b.WriteString(")")
		}
		b.WriteString(": ")
	}
	b.WriteString(e.Kind.String())
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

// Unwrap lets errors.Is and errors.As see the cause.
func (e *Error) Unwrap() error { return e.Err }

// KindOf returns the Kind of the first *Error in err's chain. Context
// errors without an *Error are recognized too.
func KindOf(err error) Kind {
	var e *Error
	switch {
	case err == nil:
		return KindUnknown
	case errors.As(err, &e):
		return e.Kind
	case errors.Is(err, context.Canceled):
		return KindCanceled
	default:
		return KindUnknown
	}
}

// ExitCode maps a kind to the msh exit code.
func ExitCode(k Kind) exitcode.Code {
	switch k {
	case KindUsage:
		return exitcode.Usage
	case KindConfig, KindInventory, KindTask, KindTaskArgs:
		return exitcode.Config
	case KindDNS, KindTCP, KindHandshake, KindAuth, KindHostKey:
		return exitcode.Connection
	case KindConnectTimeout, KindRemoteTimeout, KindGlobalTimeout:
		return exitcode.Timeout
	case KindRemoteCommand:
		return exitcode.Failure
	case KindScriptTransfer, KindBinaryTransfer:
		return exitcode.Transfer
	case KindCanceled:
		return exitcode.Canceled
	case KindTmuxUnavailable, KindTmuxStartup:
		return exitcode.Tmux
	case KindFileRead, KindFileWrite:
		return exitcode.IO
	default:
		return exitcode.Internal
	}
}
