package unixfd

import (
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// ConnectAndListen establishes a connection to the odiglet's Unix socket and retrieves a file descriptor (FD)
// for the specified eBPF map type. The provided onFD callback is invoked only when a new FD is received,
// which happens if odiglet is restarted and creates a new map (resulting in a different FD).
func ConnectAndListen(ctx context.Context, socketPath string, requestType string, logger *zap.Logger, onFD func(fd int)) error {
	var lastFD = -1

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to connect and get FD
		fd, err := connectAndGetFD(ctx, socketPath, requestType)
		if err != nil {
			// Connection attempt failed—odiglet may be down or in the process of restarting.
			// Retry the connection after a short delay.
			sleepTime := 2 * time.Second
			logger.Info("Waiting for odiglet unix socket to be ready to receive FD", zap.Error(err), zap.Duration("sleepTime", sleepTime))
			time.Sleep(sleepTime)
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

// connectAndGetFD makes a single connection, gets one FD, and closes the connection.
func connectAndGetFD(ctx context.Context, socketPath string, requestType string) (int, error) {
	fds, err := connectAndGetFDs(ctx, socketPath, requestType)
	if err != nil {
		return -1, err
	}
	if len(fds) != 1 {
		return -1, fmt.Errorf("expected 1 fd, got %d", len(fds))
	}
	return fds[0], nil
}

// ConnectAndListenMulti is like ConnectAndListen but receives multiple file descriptors
// in a single message. The onFDs callback is invoked only when the set of FDs changes.
func ConnectAndListenMulti(ctx context.Context, socketPath string, requestType string, logger *zap.Logger, onFDs func(fds []int)) error {
	var lastFDs []int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fds, err := connectAndGetFDs(ctx, socketPath, requestType)
		if err != nil {
			sleepTime := 2 * time.Second
			logger.Info("Waiting for odiglet unix socket to be ready to receive FDs", zap.Error(err), zap.Duration("sleepTime", sleepTime))
			time.Sleep(sleepTime)
			continue
		}

		if !slices.Equal(fds, lastFDs) {
			lastFDs = fds
			onFDs(fds)
		}

		if err := waitForSocketChange(ctx, socketPath); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastFDs = nil
			continue
		}
	}
}

// connectAndGetFDs makes a single connection, gets multiple FDs, and closes connection
func connectAndGetFDs(ctx context.Context, socketPath string, requestType string) ([]int, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	if _, err := conn.Write([]byte(requestType)); err != nil {
		return nil, err
	}

	return recvFDs(conn.(*net.UnixConn))
}

// recvFDs reads multiple file descriptors from the connection
func recvFDs(c *net.UnixConn) ([]int, error) {
	buf := make([]byte, 16)
	oob := make([]byte, unix.CmsgSpace(4*8)) // space for up to 8 FDs

	n, oobn, _, _, err := c.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, fmt.Errorf("readmsg: %w", err)
	}

	if n > 0 {
		response := string(buf[:n])
		if response != RespOK {
			return nil, fmt.Errorf("unexpected server response: %q, expected %q", response, RespOK)
		}
	}

	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, fmt.Errorf("parse scm: %w", err)
	}
	if len(msgs) != 1 {
		return nil, fmt.Errorf("expected 1 control message, got %d", len(msgs))
	}

	fds, err := unix.ParseUnixRights(&msgs[0])
	if err != nil {
		return nil, fmt.Errorf("parse rights: %w", err)
	}
	if len(fds) == 0 {
		return nil, fmt.Errorf("no fds received")
	}

	return fds, nil
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
