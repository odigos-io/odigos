package server

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// New builds the cluster MCP HTTP server. Phase 0 exposes a single `ping`
// tool. Real cluster tools (source/collector/destination) land in Phase 1.
func New() (*mcpserver.StreamableHTTPServer, error) {
	mcpServer := mcpserver.NewMCPServer(
		"odigos-agent-cluster-mcp",
		"0.0.1",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithRecovery(),
	)

	registerPing(mcpServer)

	httpServer := mcpserver.NewStreamableHTTPServer(
		mcpServer,
		mcpserver.WithEndpointPath("/mcp"),
	)
	return httpServer, nil
}

func registerPing(s *mcpserver.MCPServer) {
	tool := mcp.NewTool(
		"cluster_ping",
		mcp.WithDescription("Cluster MCP health check. Returns the literal string \"pong\" plus the server name and a UTC timestamp."),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(
			"pong from cluster-mcp at " + time.Now().UTC().Format(time.RFC3339),
		), nil
	})
}
