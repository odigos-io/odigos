package unixfd

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"
)

// Server serves eBPF map file descriptors to clients over a Unix socket.
// It acts as a bridge between odiglet (which creates eBPF maps) and data collection clients
// (which need access to those maps for reading trace data).
type Server struct {
	SocketPath string
	Logger     logr.Logger
	FDProvider func() int // Function that returns the current eBPF map file descriptor
}

// Run starts the Unix domain socket server, which serves eBPF map file descriptors to connecting clients.
// This allows data collection to access the eBPF map created by odiglet for reading trace data.
// The server listens for client requests and, upon receiving a request, sends the current updated eBPF map file descriptor.
func (s *Server) Run(ctx context.Context) error {
	// Remove old socket file
	_ = os.Remove(s.SocketPath)

	listener, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.SocketPath, err)
	}
	defer func() {
		_ = listener.Close()
	}()
	defer func() {
		_ = os.Remove(s.SocketPath)
	}()

	s.Logger.Info("unixfd server started", "socket", s.SocketPath)

	// Close listener when context is canceled
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				s.Logger.Info("server shutting down")
				return nil
			}
			s.Logger.Error(err, "accept failed")
			continue
		}

		go s.handleRequest(conn.(*net.UnixConn))
	}
}

// handleRequest processes a single client request
func (s *Server) handleRequest(conn *net.UnixConn) {
	defer func() {
		_ = conn.Close()
	}()

	// Read the request
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	if err != nil {
		s.Logger.Error(err, "failed to read request")
		return
	}

	request := string(buf[:n])
	if request != ReqGetFD {
		s.Logger.Info("unknown request", "request", request)
		return
	}

	// Get the current FD
	fd := s.FDProvider()
	if fd <= 0 {
		s.Logger.Error(fmt.Errorf("invalid fd %d", fd), "FDProvider returned invalid fd")
		return
	}

	// Send the FD to client
	if err := sendFD(conn, fd); err != nil {
		s.Logger.Error(err, "failed to send FD")
		return
	}

	s.Logger.Info("sent FD to client", "fd", fd)
}

// sendFD sends a file descriptor over the Unix socket
func sendFD(conn *net.UnixConn, fd int) error {
	rights := unix.UnixRights(fd)
	_, _, err := conn.WriteMsgUnix([]byte("OK"), rights, nil)
	return err
}
