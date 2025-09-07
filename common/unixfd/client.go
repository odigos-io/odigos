package unixfd

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

// ConnectAndListen connects to odiglet and gets FD only when it changes (odiglet restarts).
// onFD is called once per odiglet restart with the new FD.
func ConnectAndListen(ctx context.Context, socketPath string, onFD func(fd int)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to connect and get FD
		fd, err := connectAndGetFD(socketPath)
		if err != nil {
			// Connection failed, odiglet probably not ready yet
			time.Sleep(2 * time.Second)
			continue
		}

		// Successfully got FD - this means odiglet started/restarted
		onFD(fd)

		// Now wait for odiglet to restart by trying to connect again
		// This will block until odiglet restarts (connection succeeds again)
		waitForOdigletRestart(ctx, socketPath)
	}
}

// connectAndGetFD makes a single connection, gets FD, and closes connection
func connectAndGetFD(socketPath string) (int, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return -1, err
	}
	defer conn.Close()

	// Request the FD
	if _, err := conn.Write([]byte(ReqGetFD)); err != nil {
		return -1, err
	}

	// Receive the FD
	return recvFD(conn.(*net.UnixConn))
}

// waitForOdigletRestart waits until odiglet restarts by polling connection
func waitForOdigletRestart(ctx context.Context, socketPath string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
			// Try to connect - if it works, odiglet restarted
			conn, err := net.Dial("unix", socketPath)
			if err == nil {
				conn.Close()
				return // Odiglet is back, exit wait loop
			}
			// Still down, keep waiting
		}
	}
}

// recvFD reads a file descriptor from the connection
func recvFD(c *net.UnixConn) (int, error) {
	buf := make([]byte, 16)
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
