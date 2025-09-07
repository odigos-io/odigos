package unixfd

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// ConnectAndListen establishes a connection to the odiglet's Unix socket and retrieves a file descriptor (FD)
// for the eBPF map used for tracing. The provided onFD callback is invoked only when a new FD is received,
// which happens if odiglet is restarted and creates a new map (resulting in a different FD).
func ConnectAndListen(ctx context.Context, socketPath string, onFD func(fd int)) error {
	var lastFD int = -1

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to connect and get FD
		fd, err := connectAndGetFD(socketPath)
		if err != nil {
			// Connection attempt failed—odiglet may be down or in the process of restarting.
			// Retry the connection after a short delay.
			time.Sleep(2 * time.Second)
			continue
		}

		if fd != lastFD {
			// This is either the first time we're getting an FD,
			// or we got a different FD (indicating odiglet restarted with a new map)
			lastFD = fd
			onFD(fd)
		}

		// After getting the FD, monitor the socket file for changes
		// This allows us to detect odiglet restarts without polling
		// Once socket is changed, we reset the lastFD to -1 and continue the loop
		// This will cause the client to reconnect and get a new FD
		if err := waitForSocketChange(ctx, socketPath); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastFD = -1
			continue
		}
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

// waitForSocketChange monitors the socket file and returns when it changes or disappears
// This allows us to detect odiglet restarts without continuous polling
func waitForSocketChange(ctx context.Context, socketPath string) error {
	// Get initial file info
	initialStat, err := os.Stat(socketPath)
	if err != nil {
		// Socket doesn't exist, odiglet probably restarted
		return fmt.Errorf("socket disappeared: %w", err)
	}

	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if socket still exists and hasn't changed
			currentStat, err := os.Stat(socketPath)
			if err != nil {
				// Socket disappeared, odiglet restarted
				return fmt.Errorf("socket disappeared: %w", err)
			}

			// Check if the socket file changed (different inode or modification time)
			if !os.SameFile(initialStat, currentStat) ||
				currentStat.ModTime() != initialStat.ModTime() {
				// Socket changed, odiglet likely restarted
				return fmt.Errorf("socket changed")
			}

		}
	}
}
