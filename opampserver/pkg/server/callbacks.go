package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Callbacks is an implementation of the Callbacks interface.
var _ types.Callbacks = &K8sCrdCallbacks{}

type K8sCrdCallbacks struct {
	logger     logr.Logger
	kubeclient client.Client
}

func (c *K8sCrdCallbacks) OnConnecting(request *http.Request) types.ConnectionResponse {
	deviceId, err := getDeviceIdFromHeader(request)
	if err != nil {
		return types.ConnectionResponse{
			Accept:         false,
			HTTPStatusCode: 401,
		}
	}

	println("OnConnecting", deviceId)
	return types.ConnectionResponse{
		ConnectionCallbacks: &ConnectionCallbacks{
			kubeclient: c.kubeclient,
		},
		Accept:         true,
		HTTPStatusCode: 200,
	}
}

// ConnectionCallbacks is an implementation of the ConnectionCallbacks interface from opamp server.
var _ types.ConnectionCallbacks = &ConnectionCallbacks{}

type ConnectionCallbacks struct {
	kubeclient client.Client
}

func (c *ConnectionCallbacks) OnConnected(ctx context.Context, conn types.Connection) {
	println("OnConnected")
}

func (c *ConnectionCallbacks) OnMessage(ctx context.Context, conn types.Connection, message *protobufs.AgentToServer) *protobufs.ServerToAgent {
	println("OnMessage", message.String())
	return &protobufs.ServerToAgent{}
}

func (c *ConnectionCallbacks) OnConnectionClose(conn types.Connection) {
	println("OnConnectionClose")
}

func getDeviceIdFromHeader(request *http.Request) (string, error) {
	authorization := request.Header.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("Authorization header is missing")
	}

	// make sure the Authorization header is in the format "DeviceId <device-id>"
	const prefix = "DeviceId "
	if len(authorization) <= len(prefix) || authorization[:len(prefix)] != prefix {
		return "", fmt.Errorf("Authorization header is not in the format 'DeviceId <device-id>'")
	}

	return authorization[len(prefix):], nil
}
