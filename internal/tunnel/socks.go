package tunnel

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"time"
)

// SOCKS5 constants from RFC 1928.
const (
	socksVersion = 5
	noAuth       = 0
	noMethod     = 0xFF
	cmdConnect   = 1
	addrIPv4     = 1
	addrName     = 3
	addrIPv6     = 4

	replyOK          = 0
	replyUnreachable = 4
	replyBadCommand  = 7
	replyBadAddress  = 8
)

// handshakeTimeout limits a client that connects and says nothing.
const handshakeTimeout = 10 * time.Second

// DialAddr opens a connection to an address chosen by the client.
type DialAddr func(ctx context.Context, addr string) (net.Conn, error)

// Socks returns a Handler that speaks SOCKS5 and connects with dial.
func Socks(dial DialAddr, report func(error)) Handler {
	return func(ctx context.Context, conn net.Conn) {
		_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
		target, err := socksRequest(conn)
		if err != nil {
			return // a client that doesn't speak SOCKS5 is not worth a message
		}
		far, err := dial(ctx, target)
		if err != nil {
			socksReply(conn, replyUnreachable)
			if ctx.Err() == nil {
				report(err)
			}
			return
		}
		if socksReply(conn, replyOK) != nil {
			far.Close()
			return
		}
		_ = conn.SetDeadline(time.Time{})
		Pipe(ctx, conn, far)
	}
}

// socksRequest reads the greeting and the request and returns host:port.
func socksRequest(conn net.Conn) (string, error) {
	var head [2]byte
	if _, err := io.ReadFull(conn, head[:]); err != nil {
		return "", err
	}
	if head[0] != socksVersion {
		return "", errors.New("not SOCKS5")
	}
	methods := make([]byte, head[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}
	if !slices.Contains(methods, noAuth) {
		conn.Write([]byte{socksVersion, noMethod})
		return "", errors.New("the client needs authentication")
	}
	if _, err := conn.Write([]byte{socksVersion, noAuth}); err != nil {
		return "", err
	}

	var req [4]byte // version, command, reserved, address type
	if _, err := io.ReadFull(conn, req[:]); err != nil {
		return "", err
	}
	if req[0] != socksVersion {
		return "", errors.New("not SOCKS5")
	}
	if req[1] != cmdConnect {
		socksReply(conn, replyBadCommand)
		return "", errors.New("only CONNECT is supported")
	}
	host, err := socksHost(conn, req[3])
	if err != nil {
		return "", err
	}
	var port [2]byte
	if _, err := io.ReadFull(conn, port[:]); err != nil {
		return "", err
	}
	return net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(port[:])))), nil
}

// socksHost reads an address of the given type.
func socksHost(conn net.Conn, kind byte) (string, error) {
	size := 0
	switch kind {
	case addrIPv4:
		size = net.IPv4len
	case addrIPv6:
		size = net.IPv6len
	case addrName:
		var n [1]byte
		if _, err := io.ReadFull(conn, n[:]); err != nil {
			return "", err
		}
		size = int(n[0])
	default:
		socksReply(conn, replyBadAddress)
		return "", fmt.Errorf("unknown address type %d", kind)
	}
	raw := make([]byte, size)
	if _, err := io.ReadFull(conn, raw); err != nil {
		return "", err
	}
	if kind == addrName {
		return string(raw), nil
	}
	return net.IP(raw).String(), nil
}

// socksReply answers a request. The bound address is not used by clients
// of CONNECT, so it is all zeros.
func socksReply(conn net.Conn, code byte) error {
	_, err := conn.Write([]byte{socksVersion, code, 0, addrIPv4, 0, 0, 0, 0, 0, 0})
	return err
}
