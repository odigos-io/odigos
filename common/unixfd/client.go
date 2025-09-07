package unixfd

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// RequestFD connects to the server, sends GET_FD, and receives the FD.
// It enforces the NEW_FD handshake: first wait for MsgNewFD, then receive FD_SENT + FD.
func RequestFD(socketPath string) (int, error) {
	raddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return -1, fmt.Errorf("resolve unix addr: %w", err)
	}

	conn, err := net.DialUnix("unix", nil, raddr)
	if err != nil {
		return -1, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Send request
	if _, err := conn.Write([]byte(ReqGetFD)); err != nil {
		return -1, fmt.Errorf("write request: %w", err)
	}

	// --- Step 1: expect NEW_FD ---
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	if err != nil {
		return -1, fmt.Errorf("read NEW_FD: %w", err)
	}
	if string(buf[:n]) != MsgNewFD {
		return -1, fmt.Errorf("expected %q, got %q", MsgNewFD, string(buf[:n]))
	}

	// --- Step 2: receive FD ---
	fd, err := recvFD(conn)
	if err != nil {
		return -1, fmt.Errorf("recvFD: %w", err)
	}
	return fd, nil
}

// recvFD extracts the file descriptor from the control message.
func recvFD(c *net.UnixConn) (int, error) {
	buf := make([]byte, 16)
	oob := make([]byte, unix.CmsgSpace(4))

	n, oobn, _, _, err := c.ReadMsgUnix(buf, oob)
	if err != nil {
		return -1, fmt.Errorf("readmsg: %w", err)
	}

	if string(buf[:n]) != MsgFDSent {
		return -1, fmt.Errorf("expected %q, got %q", MsgFDSent, string(buf[:n]))
	}

	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, fmt.Errorf("parse scm: %w", err)
	}
	if len(msgs) != 1 {
		return -1, fmt.Errorf("expected 1 control message, got %d", len(msgs))
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
