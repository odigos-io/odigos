package unixfd

import (
	"context"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

// Server serves eBPF map file descriptors to clients over a Unix socket.
// It acts as a bridge between odiglet (which creates eBPF maps) and data collection clients
// (which need access to those maps for reading telemetry data).
type Server struct {
	SocketPath         string
	TracesFDProvider   func() int   // Function that returns the traces eBPF map file descriptor
	MetricsFDsProvider func() []int // Function that returns all metrics-related eBPF map file descriptors
}

// Run starts the Unix domain socket server, which serves eBPF map file descriptors to connecting clients.
// This allows data collection to access the eBPF map created by odiglet for reading trace data.
// The server listens for client requests and, upon receiving a request, sends the current updated eBPF map file descriptor.
func (s *Server) Run(ctx context.Context) error {
	// Remove old socket file
	_ = os.Remove(s.SocketPath)
	var lc net.ListenConfig

	listener, err := lc.Listen(ctx, "unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.SocketPath, err)
	}
	defer func() {
		_ = listener.Close()
	}()
	defer func() {
		_ = os.Remove(s.SocketPath)
	}()

	logger := commonlogger.Logger().With("subsystem", "unixfd")
	logger.Info("unixfd server started", "socket", s.SocketPath)

	// Close listener when context is canceled
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("server shutting down")
				return nil
			}
			logger.Error("accept failed", "err", err)
			continue
		}

		go s.handleRequest(conn.(*net.UnixConn))
	}
}

// handleRequest processes a single client request
func (s *Server) handleRequest(conn *net.UnixConn) {
	logger := commonlogger.Logger().With("subsystem", "unixfd")
	defer func() {
		_ = conn.Close()
	}()

	// Read the request
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	if err != nil {
		logger.Error("failed to read request", "err", err)
		return
	}

	request := string(buf[:n])

	switch request {
	case ReqGetFD, ReqGetTracesFD:
		// Legacy "GET_FD" defaults to traces for backward compatibility
		if s.TracesFDProvider == nil {
			logger.Error("no traces FD provider", "err", fmt.Errorf("traces FD provider not configured"))
			return
		}
		fd := s.TracesFDProvider()
		if fd <= 0 {
			logger.Error("FD provider returned invalid fd", "err", fmt.Errorf("invalid fd %d", fd), "request", request)
			return
		}
		if err := sendFDs(conn, fd); err != nil {
			logger.Error("failed to send FD", "err", err)
			return
		}
		logger.Info("sent FD to client", "fd", fd, "request", request)

	case ReqGetMetricsFD:
		if s.MetricsFDsProvider == nil {
			logger.Error("no metrics FDs provider", "err", fmt.Errorf("metrics FDs provider not configured"))
			return
		}
		fds := s.MetricsFDsProvider()
		if len(fds) == 0 {
			logger.Error("metrics FDs provider returned empty slice", "err", fmt.Errorf("no metrics FDs available"))
			return
		}
		for _, fd := range fds {
			if fd <= 0 {
				logger.Error("FD provider returned invalid fd", "err", fmt.Errorf("invalid fd %d", fd), "request", request)
				return
			}
		}
		if err := sendFDs(conn, fds...); err != nil {
			logger.Error("failed to send metrics FDs", "err", err)
			return
		}
		logger.Info("sent metrics FDs to client", "fds", fds, "request", request)

	default:
		logger.Info("unknown request", "request", request)
		return
	}
}

// sendFDs sends one or more file descriptors over the Unix socket in a single message.
func sendFDs(conn *net.UnixConn, fds ...int) error {
	rights := unix.UnixRights(fds...)
	_, _, err := conn.WriteMsgUnix([]byte(RespOK), rights, nil)
	return err
}
