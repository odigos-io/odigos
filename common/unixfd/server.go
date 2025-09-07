package unixfd

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"
)

// Server handles Unix socket FD requests from data-collection.
type Server struct {
	SocketPath string      // e.g. /var/exchange/exchange.sock
	Logger     logr.Logger // structured logger
	FDProvider func() int  // callback returning FD to send
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

	switch req {
	case ReqGetFD:
		fd := s.FDProvider()
		if fd <= 0 {
			s.Logger.Info("invalid FD from provider", "socket", s.SocketPath)
			return
		}
		// Always send NEW_FD + FD_SENT
		if _, err := conn.Write([]byte(MsgNewFD)); err != nil {
			s.Logger.Error(err, "failed to send NEW_FD", "socket", s.SocketPath)
			return
		}
		if err := sendFD(conn, fd); err != nil {
			s.Logger.Error(err, "sendFD failed", "socket", s.SocketPath)
		} else {
			s.Logger.Info("✅ NEW_FD + FD sent to client")
		}
	default:
		s.Logger.Info("unknown request", "req", req, "socket", s.SocketPath)
	}
}

func sendFD(c *net.UnixConn, fd int) error {
	controlMessage := unix.UnixRights(fd)
	_, _, err := c.WriteMsgUnix([]byte(MsgFDSent), controlMessage, nil)
	return err
}
