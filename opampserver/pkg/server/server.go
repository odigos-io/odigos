package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/opampserver/pkg/deviceid"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"google.golang.org/protobuf/proto"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getDeviceIdFromHeader(request *http.Request) (string, error) {
	authorization := request.Header.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	// make sure the Authorization header is in the format "DeviceId <device-id>"
	const prefix = "DeviceId "
	if len(authorization) <= len(prefix) || authorization[:len(prefix)] != prefix {
		return "", fmt.Errorf("authorization header is not in the format 'DeviceId <device-id>'")
	}

	return authorization[len(prefix):], nil
}

func StartOpAmpServer(ctx context.Context, logger logr.Logger, mgr ctrl.Manager, kubeClient *kubernetes.Clientset) error {

	listenEndpoint := "0.0.0.0:4320"
	logger.Info("Starting opamp server", "listenEndpoint", listenEndpoint)

	deviceidCache, err := deviceid.NewDeviceIdCache(logger, kubeClient)
	if err != nil {
		return err
	}

	connectionCache := NewConnectionsCache()

	handlers := &ConnectionHandlers{
		logger:        logger,
		deviceIdCache: deviceidCache,
	}

	http.HandleFunc("/v1/opamp", func(w http.ResponseWriter, req *http.Request) {
		// Check for the correct method, e.g., GET
		if req.Method != "POST" {
			http.Error(w, "Method is not supported.", http.StatusMethodNotAllowed)
			return
		}

		// we only support plain http connections.
		// this check will filter out WS connections if they arrive for any reasons.
		if req.Header.Get("Content-Type") != "application/x-protobuf" {
			http.Error(w, "Content-Type header is not application/x-protobuf", http.StatusBadRequest)
			return
		}

		bytes, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		// assume the data is not compressed, which is not relevant and not supported in odigos

		// Decode the message as a Protobuf message.
		var opampRequest protobufs.AgentToServer
		err = proto.Unmarshal(bytes, &opampRequest)
		if err != nil {
			logger.Error(err, "Cannot decode opamp message from HTTP Body")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		deviceId, err := getDeviceIdFromHeader(req)
		if err != nil {
			logger.Error(err, "Failed to get device id from header")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var opampResponse *protobufs.ServerToAgent
		connectionInfo, exists := connectionCache.GetConnection(deviceId)
		if !exists {
			connectionInfo, opampResponse, err = handlers.OnNewConnection(ctx, deviceId, &opampRequest)
			if err != nil {
				logger.Error(err, "Failed to process new connection")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			connectionCache.AddConnection(deviceId, connectionInfo)
		} else {
			opampResponse, err = handlers.OnAgentToServerMessage(ctx, &opampRequest, connectionInfo)

			if err != nil {
				logger.Error(err, "Failed to process opamp message")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		if opampResponse == nil {
			logger.Error(err, "No response from opamp handler")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// keep record in memory of last message time, to detect stale connections
		connectionCache.RecordMessageTime(deviceId)

		opampResponse.InstanceUid = opampRequest.InstanceUid

		// Marshal the response.
		bytes, err = proto.Marshal(opampResponse)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Send the response.
		w.Header().Set("Content-Type", "application/x-protobuf")
		_, err = w.Write(bytes)

		if err != nil {
			logger.Error(err, "Failed to write response")
		}
	})

	go func() {
		if err := http.ListenAndServe(listenEndpoint, nil); err != nil {
			logger.Error(err, "Error starting opamp server")
		}
	}()

	go func() {
		ticker := time.NewTicker(HeartbeatInterval)
		defer ticker.Stop() // Clean up when done
		for {
			select {
			case <-ctx.Done():
				logger.Info("Shutting down live connections timeout monitor")
				return
			case <-ticker.C:
				// Clean up stale connections
				deadConnections := connectionCache.CleanupStaleConnections()
				for _, conn := range deadConnections {
					handlers.OnConnectionClosed(ctx, &conn)
				}
			}
		}
	}()

	return nil
}
