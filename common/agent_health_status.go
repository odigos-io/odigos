package common

type AgentHealthStatus string

const (
	// AgentHealthStatusHealthy represents the healthy status of an agent
	// It started the OpenTelemetry SDK with no errors, processed any configuration and is ready to receive data.
	AgentHealthStatusHealthy AgentHealthStatus = "Healthy"

	// AgentHealthStatusUnsupportedRuntimeVersion represents that the agent is running on an unsupported runtime version
	// For example: Otel sdk supports node.js >= 14 and workload is running with node.js 12
	AgentHealthStatusUnsupportedRuntimeVersion = "UnsupportedRuntimeVersion"

	// AgentHealthStatusNoHeartbeat is when the server did not receive a 3 heartbeats from the agent, thus it is considered unhealthy
	AgentHealthStatusNoHeartbeat = "NoHeartbeat"
)
