// Package transport exposes the shared HTTP handler that backs OpAMP over both
// TCP and the node-local unix socket. The wire protocol is plain HTTP/1.1 with
// a protobuf body in both cases; only the listener type differs.
package transport

import (
	"context"
	"io"
	"net/http"

	"google.golang.org/protobuf/proto"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	commonopamp "github.com/odigos-io/odigos/common/opamp"
	"github.com/odigos-io/odigos/opampserver/pkg/opamptypes"
	"github.com/odigos-io/odigos/opampserver/protobufs"
)

// NewHandler returns a http.Handler that decodes an AgentToServer protobuf, runs
// processor.Process, and writes the ServerToAgent reply. The transport argument
// is only used as a label for downstream logging (HTTP vs Unix).
func NewHandler(ctx context.Context, processor opamptypes.Processor, transport commonopamp.OpAmpTransport) http.Handler {
	logger := commonlogger.LoggerCompat().With("subsystem", "opampserver", "transport", string(transport))

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
			logger.Error("Cannot decode opamp message", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Pass the long-lived server ctx, not req.Context(): instance updates are queued async
		// and must outlive the HTTP request (req.Context is cancelled when the response is written).
		serverToAgent, status := processor.Process(ctx, &agentToServer, transport)
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
			logger.Error("Failed to write response", "err", err)
		}
	})
	return mux
}
