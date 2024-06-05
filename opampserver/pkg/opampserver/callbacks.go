package opampserver

import (
	"context"
	"net/http"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
)

// Callbacks is an implementation of the Callbacks interface.
var _ types.Callbacks = &K8sCrdCallbacks{}

type K8sCrdCallbacks struct {
}

func (c *K8sCrdCallbacks) OnConnecting(request *http.Request) types.ConnectionResponse {
	println("OnConnecting")
	return types.ConnectionResponse{
		ConnectionCallbacks: &ConnectionCallbacks{},
		Accept:              true,
		HTTPStatusCode:      200,
	}
}

// ConnectionCallbacks is an implementation of the ConnectionCallbacks interface from opamp server.
var _ types.ConnectionCallbacks = &ConnectionCallbacks{}

type ConnectionCallbacks struct {
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
