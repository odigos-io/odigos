package server

import (
	"context"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/odigos-io/odigos-agent/mcp/tools"
)

// New builds the cluster MCP HTTP server. Phase 1 wires real cluster-state
// tools alongside the phase-0 `cluster_ping` smoke tool. Tool groups are
// added in this order: ping, source/instrumentation, (collector / destination
// / citation follow in later commits).
func New() (*mcpserver.StreamableHTTPServer, error) {
	mcpServer := mcpserver.NewMCPServer(
		"odigos-agent-cluster-mcp",
		"0.1.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithRecovery(),
	)

	registerPing(mcpServer)

	clients, err := tools.BuildClients()
	if err != nil {
		// Don't crash the process: phase 0 contracts say `cluster_ping`
		// must keep working even without kube creds (local smoke tests
		// run via docker compose without a kube context). Surface the
		// failure on every tool call instead so the user sees why.
		log.Printf("kube clients unavailable, cluster tools will return errors: %v", err)
	} else {
		approvalCache := tools.NewApprovalCache(0)
		tools.RegisterSourceTools(mcpServer, clients, approvalCache)
		tools.RegisterCollectorTools(mcpServer, clients)
		tools.RegisterDestinationTools(mcpServer, clients)
	}

	// Citation tool runs without kube creds - it only talks to
	// raw.githubusercontent.com. Register it unconditionally so the agent
	// can still cite source even if kube clients are unavailable.
	tools.RegisterCitationTools(mcpServer)

	httpServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"),
	)
	return httpServer, nil
}

func registerPing(server *mcpserver.MCPServer) {
	tool := mcp.NewTool(
		"cluster_ping",
		mcp.WithDescription("Cluster MCP health check. Returns the literal string \"pong\" plus the server name and a UTC timestamp."),
	)
	server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(
			"pong from cluster-mcp at " + time.Now().UTC().Format(time.RFC3339),
		), nil
	})
}
