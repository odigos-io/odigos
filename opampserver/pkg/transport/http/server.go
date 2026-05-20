package httptransport

import (
	"context"
	"fmt"
	"io"
	"net/http"

	commonconsts "github.com/odigos-io/odigos/common/consts"
	commonopamp "github.com/odigos-io/odigos/common/opamp"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/opampserver/pkg/opamptypes"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"google.golang.org/protobuf/proto"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Start(ctx context.Context, processor opamptypes.Processor) error {
	listenAddr := fmt.Sprintf("0.0.0.0:%d", commonconsts.OpAMPPort)
	logger := commonlogger.LoggerCompat().With("subsystem", "opampserver", "transport", "http")
	logger.Info("Starting opamp HTTP server", "listenEndpoint", listenAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/opamp", func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/x-protobuf" {
			http.Error(w, "Content-Type header is not application/x-protobuf", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var agentToServer protobufs.AgentToServer
		if err := proto.Unmarshal(body, &agentToServer); err != nil {
			logger.Error("Cannot decode opamp message from HTTP body", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		serverToAgent, status := processor.Process(req.Context(), &agentToServer, commonopamp.OpAmpTransportHTTP)
		if status != opamptypes.ProcessOK {
			if status == opamptypes.ProcessBadRequest {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		respBytes, err := proto.Marshal(serverToAgent)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-protobuf")
		if _, err := w.Write(respBytes); err != nil {
			logger.Error("Failed to write HTTP response", "err", err)
		}
	})

	httpServer := &http.Server{Addr: listenAddr, Handler: mux}

	go func() {
		<-ctx.Done()
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error("Failed to shut down HTTP opamp server", "err", err)
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
