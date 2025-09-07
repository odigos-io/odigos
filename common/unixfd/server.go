package unixfd

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"
)

// Server handles Unix socket FD requests from data-collection.
type Server struct {
	SocketPath string
	Logger     logr.Logger
	FDProvider func() int

	mu    sync.Mutex
	conns []*net.UnixConn // active client connections
}

// Run starts listening and serving requests until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	_ = os.Remove(s.SocketPath)

	addr, err := net.ResolveUnixAddr("unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("resolve unix addr: %w", err)
	}

	ul, err := net.ListenUnix("unix", addr)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	defer ul.Close()

	s.Logger.Info("unixfd server listening", "socket", s.SocketPath)

	for {
		select {
		case <-ctx.Done():
			s.Logger.Info("unixfd server shutting down")
			return nil
		default:
		}

		conn, err := ul.AcceptUnix()
		if err != nil {
			s.Logger.Error(err, "accept failed")
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn *net.UnixConn) {
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	if err != nil {
		conn.Close()
		return
	}
	req := string(buf[:n])

	if req == ReqGetFD {
		fd := s.FDProvider()
		if fd <= 0 {
			s.Logger.Error(fmt.Errorf("invalid fd"), "FDProvider returned invalid fd")
			conn.Close()
			return
		}

		// Track connection for future NEW_FD pushes
		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()

		// Reply with FD_SENT
		if err := sendFD(conn, fd, MsgFDSent); err != nil {
			s.Logger.Error(err, "sendFD failed")
		} else {
			s.Logger.Info("✅ FD sent to client", "fd", fd)
		}
	} else {
		s.Logger.Info("unknown request", "req", req)
		conn.Close()
	}
}

// NotifyNewFD pushes a NEW_FD message to all active clients.
// Called when odiglet restarts or recreates its map.
func (s *Server) NotifyNewFD() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fd := s.FDProvider()
	if fd <= 0 {
		s.Logger.Error(fmt.Errorf("invalid fd"), "FDProvider returned invalid fd for NotifyNewFD")
		return
	}

	for _, c := range s.conns {
		if err := sendFD(c, fd, MsgNewFD); err != nil {
			s.Logger.Error(err, "failed to push NEW_FD, dropping client")
			c.Close()
		} else {
			s.Logger.Info("pushed NEW_FD to client", "fd", fd)
		}
	}
}

// sendFD sends a single file descriptor with a message.
func sendFD(c *net.UnixConn, fd int, msg string) error {
	control := unix.UnixRights(fd)
	_, _, err := c.WriteMsgUnix([]byte(msg), control, nil)
	return err
}
