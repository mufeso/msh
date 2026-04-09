// Package exitcode defines the process exit codes of msh.
//
// Scripts that call msh can rely on these values:
//
//	0   OK          everything succeeded
//	1   Failure     a remote command ran and exited non-zero
//	2   Usage       invalid command-line input
//	3   Config      invalid configuration, inventory, task or task arguments
//	4   Connection  DNS, TCP, SSH handshake, authentication, host key or bastion failure
//	5   Timeout     a command timeout or the global timeout expired
//	6   Transfer    a script or binary could not be transferred
//	7   Tmux        tmux is unavailable or failed
//	8   IO          a local file could not be read or written
//	70  Internal    an unexpected internal error (a bug in msh)
//	130 Canceled    the user pressed CTRL+C (128 + SIGINT)
package exitcode

// Code is a process exit code. Because it is a distinct type, a Code
// cannot be mixed up with an ordinary int by accident.
type Code int

// The exit codes of msh.
const (
	OK         Code = 0
	Failure    Code = 1
	Usage      Code = 2
	Config     Code = 3
	Connection Code = 4
	Timeout    Code = 5
	Transfer   Code = 6
	Tmux       Code = 7
	IO         Code = 8
	Internal   Code = 70
	Canceled   Code = 130
)

// String implements fmt.Stringer, so fmt prints Codes by name.
func (c Code) String() string {
	switch c {
	case OK:
		return "ok"
	case Failure:
		return "failure"
	case Usage:
		return "usage"
	case Config:
		return "config"
	case Connection:
		return "connection"
	case Timeout:
		return "timeout"
	case Transfer:
		return "transfer"
	case Tmux:
		return "tmux"
	case IO:
		return "io"
	case Internal:
		return "internal"
	case Canceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// Description is a documented exit code.
type Description struct {
	Code    Code
	Meaning string
}

// All returns every exit code with its meaning, in numeric order.
func All() []Description {
	return []Description{
		{OK, "every selected server succeeded"},
		{Failure, "a remote command ran and exited non-zero"},
		{Usage, "invalid command-line input"},
		{Config, "invalid configuration, inventory, task or task arguments"},
		{Connection, "DNS, TCP, SSH handshake, authentication, host key or bastion failure"},
		{Timeout, "a connection, command or the global timeout expired"},
		{Transfer, "a script could not be transferred"},
		{Tmux, "tmux is unavailable or failed"},
		{IO, "a local file could not be read or written"},
		{Internal, "an unexpected internal error (a bug in msh)"},
		{Canceled, "interrupted by CTRL+C, SIGTERM or SIGHUP"},
	}
}
