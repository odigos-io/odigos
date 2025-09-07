package unixfd

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// RequestFD connects to the Unix socket, requests an FD, and returns it.
func RequestFD(socketPath string) (int, error) {
	raddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return -1, fmt.Errorf("resolve unix addr: %w", err)
	}

	conn, err := net.DialUnix("unix", nil, raddr)
	if err != nil {
		return -1, fmt.Errorf("dial unix: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("GET_FD")); err != nil {
		return -1, fmt.Errorf("write request: %w", err)
	}

	// Receive FD
	return recvFD(conn)
}

func recvFD(c *net.UnixConn) (int, error) {
	buf := make([]byte, 1)
	oob := make([]byte, unix.CmsgSpace(4))

	_, oobn, _, _, err := c.ReadMsgUnix(buf, oob)
	if err != nil {
		return -1, fmt.Errorf("readmsg: %w", err)
	}

	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, fmt.Errorf("parse scm: %w", err)
	}

	if len(msgs) != 1 {
		return -1, fmt.Errorf("expected 1 control message got %d", len(msgs))
	}

	fds, err := unix.ParseUnixRights(&msgs[0])
	if err != nil {
		return -1, fmt.Errorf("parse rights: %w", err)
	}
	if len(fds) == 0 {
		return -1, fmt.Errorf("no fd received")
	}

	return fds[0], nil
}
