package sshx

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"

	"example.com/msh/internal/errs"
)

// Forward asks the bastion to open a TCP connection to ep (an SSH
// "direct-tcpip" channel). The returned net.Conn carries the bytes through
// the encrypted bastion connection; nothing is executed on the bastion.
//
// Because the result is an ordinary net.Conn, the SSH handshake with the
// target can run over it exactly as over a direct TCP connection.
func Forward(ctx context.Context, bastion *ssh.Client, bastionName string, ep Endpoint) (net.Conn, error) {
	conn, err := bastion.DialContext(ctx, "tcp", ep.Addr())
	if err != nil {
		if ctx.Err() != nil {
			return nil, contextError(ctx, ep)
		}
		return nil, errs.New(errs.KindTCP, ep.Name, fmt.Errorf("via bastion %s: %w", bastionName, err))
	}
	return conn, nil
}
