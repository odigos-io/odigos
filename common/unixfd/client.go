package unixfd

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// ConnectAndListen connects to odiglet, requests FD once, and then blocks reading FD updates.
// The callback is invoked whenever a new FD arrives (initial or NEW_FD).
func ConnectAndListen(socketPath string, onFD func(fd int, msg string)) error {
	raddr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return fmt.Errorf("resolve unix addr: %w", err)
	}

	conn, err := net.DialUnix("unix", nil, raddr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	// Send initial GET_FD request
	if _, err := conn.Write([]byte(ReqGetFD)); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	go func() {
		defer conn.Close()

		for {
			fd, msg, err := recvFD(conn)
			if err != nil {
				return // connection closed, odiglet probably restarted
			}
			onFD(fd, msg)
		}
	}()

	return nil
}

// recvFD reads a message + FD from the server.
func recvFD(c *net.UnixConn) (int, string, error) {
	buf := make([]byte, 16)
	oob := make([]byte, unix.CmsgSpace(4))

	n, oobn, _, _, err := c.ReadMsgUnix(buf, oob)
	if err != nil {
		return -1, "", fmt.Errorf("readmsg: %w", err)
	}
	msg := string(buf[:n])

	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, "", fmt.Errorf("parse scm: %w", err)
	}
	if len(msgs) != 1 {
		return -1, "", fmt.Errorf("expected 1 control message, got %d", len(msgs))
	}
	fds, err := unix.ParseUnixRights(&msgs[0])
	if err != nil {
		return -1, "", fmt.Errorf("parse rights: %w", err)
	}
	if len(fds) == 0 {
		return -1, "", fmt.Errorf("no fd received")
	}

	return fds[0], msg, nil
}
