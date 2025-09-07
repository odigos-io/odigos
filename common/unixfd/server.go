package unixfd

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"
)

// Server handles Unix socket FD requests.
// Typical usage: odiglet side, serving one or more clients (data-collection restarts).
type Server struct {
	SocketPath string // where to create the Unix socket, e.g. /var/exchange/exchange.sock
	Logger     logr.Logger
	FDProvider func() int // callback to fetch the FD to send (e.g. ebpf.Map.FD)
}

// Run starts listening and serving requests until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	_ = os.Remove(s.SocketPath) // cleanup stale file

	addr, err := net.ResolveUnixAddr("unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("resolve unix addr: %w", err)
	}

	ul, err := net.ListenUnix("unix", addr)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	s.Logger.Info("unixfd server listening", "socket", s.SocketPath)
	defer ul.Close()

	for {
		// Handle cancellation
		select {
		case <-ctx.Done():
			s.Logger.Info("unixfd server shutting down", "socket", s.SocketPath)
			return nil
		default:
		}

		// Accept new client
		conn, err := ul.AcceptUnix()
		if err != nil {
			s.Logger.Error(err, "accept failed", "socket", s.SocketPath)
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn *net.UnixConn) {
	defer conn.Close()

	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	if err != nil {
		s.Logger.Error(err, "failed to read request", "socket", s.SocketPath)
		return
	}

	req := string(buf[:n])
	s.Logger.Info("received request", "req", req, "socket", s.SocketPath)

	if req == "GET_FD" {
		fd := s.FDProvider()
		if err := sendFD(conn, fd); err != nil {
			s.Logger.Error(err, "sendFD failed", "socket", s.SocketPath)
		} else {
			s.Logger.Info("✅ FD sent to client")
		}
	} else {
		s.Logger.Info("unknown request", "req", req, "socket", s.SocketPath)
	}
}

// sendFD sends a single file descriptor over a Unix domain socket.
func sendFD(c *net.UnixConn, fd int) error {
	controlMessage := unix.UnixRights(fd)
	_, _, err := c.WriteMsgUnix([]byte("FD_SENT"), controlMessage, nil)
	return err
}
