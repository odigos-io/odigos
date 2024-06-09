package server

import (
	"context"
	"net/http"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Callbacks is an implementation of the Callbacks interface.
var _ types.Callbacks = &K8sCrdCallbacks{}

type K8sCrdCallbacks struct {
	kubeclient client.Client
}

func (c *K8sCrdCallbacks) OnConnecting(request *http.Request) types.ConnectionResponse {
	println("OnConnecting")
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
