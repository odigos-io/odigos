package unixtransport

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonopamp "github.com/odigos-io/odigos/common/opamp"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/opampserver/pkg/opamptypes"
	"github.com/odigos-io/odigos/opampserver/pkg/transport"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	SocketPath string
}

func NewServer() *Server {
	return &Server{SocketPath: k8sconsts.OdigosOpampUnixSocketPath}
}

func (s *Server) Kind() commonopamp.OpAmpTransport {
	return commonopamp.OpAmpTransportUnix
}

func (s *Server) Start(ctx context.Context, processor opamptypes.Processor) error {
	logger := commonlogger.LoggerCompat().With("subsystem", "opampserver", "transport", "unix")

	if err := os.MkdirAll(k8sconsts.OdigosOpampExchangeDir, 0o755); err != nil {
		return fmt.Errorf("mkdir exchange dir: %w", err)
	}

	_ = os.Remove(s.SocketPath)

	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.SocketPath, err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			logger.Error("Failed to close Unix opamp listener", "err", err)
		}
		if err := os.Remove(s.SocketPath); err != nil && !os.IsNotExist(err) {
			logger.Error("Failed to remove Unix opamp socket", "err", err, "socket", s.SocketPath)
		}
	}()

	if err := os.Chmod(s.SocketPath, 0o666); err != nil {
		logger.Error("Failed to chmod unix socket", "err", err, "socket", s.SocketPath)
	}

	logger.Info("Starting opamp Unix server", "socket", s.SocketPath)

	go func() {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			logger.Error("Failed to shut down Unix opamp server", "err", err)
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("Unix opamp server shutting down")
				return nil
			}
			logger.Error("accept failed", "err", err)
			continue
		}
		go s.handleConnection(ctx, conn, processor, logger)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn, processor opamptypes.Processor, logger *commonlogger.OdigosLogger) {
	defer func() { _ = conn.Close() }()

	for {
		if ctx.Err() != nil {
			return
		}

		payload, err := transport.ReadFrame(conn)
		if err != nil {
			if ctx.Err() == nil {
				logger.Debug("Unix connection read ended", "err", err)
			}
			return
		}

		var agentToServer protobufs.AgentToServer
		if err := proto.Unmarshal(payload, &agentToServer); err != nil {
			logger.Error("Cannot decode opamp message from Unix frame", "err", err)
			return
		}

		serverToAgent, status := processor.Process(ctx, &agentToServer, commonopamp.OpAmpTransportUnix)
		if status == opamptypes.ProcessBadRequest {
			return
		}
		if status == opamptypes.ProcessError || serverToAgent == nil {
			return
		}

		respBytes, err := proto.Marshal(serverToAgent)
		if err != nil {
			logger.Error("Failed to marshal Unix opamp response", "err", err)
			return
		}

		if err := transport.WriteFrame(conn, respBytes); err != nil {
			logger.Debug("Failed to write Unix opamp response", "err", err)
			return
		}
	}
}
