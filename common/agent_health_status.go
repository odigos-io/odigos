package common

type AgentHealthStatus string

const (
	// AgentHealthStatusHealthy represents the healthy status of an agent
	// It started the OpenTelemetry SDK with no errors, processed any configuration and is ready to receive data.
	AgentHealthStatusHealthy AgentHealthStatus = "Healthy"

	// AgentHealthStatusStarting represents that the agent is starting and there is still no health status available.
	// Once the agent finishes starting, it should report an either healthy or unhealthy status depending on the result.
	AgentHealthStatusStarting AgentHealthStatus = "Starting"

	// AgentHealthStatusUnsupportedRuntimeVersion represents that the agent is running on an unsupported runtime version
	// For example: Otel sdk supports node.js >= 14 and workload is running with node.js 12
	AgentHealthStatusUnsupportedRuntimeVersion AgentHealthStatus = "UnsupportedRuntimeVersion"

	// AgentHealthStatusNoHeartbeat is when the server did not receive a 3 heartbeats from the agent, thus it is considered unhealthy
	AgentHealthStatusNoHeartbeat AgentHealthStatus = "NoHeartbeat"

	// AgentHealthStatusProcessTerminated is when the agent process is terminated.
	// The termination can be due to normal shutdown (e.g. event loop run out of work)
	// due to explicit termination (e.g. code calls exit(), or OS signal), or due to an error (e.g. unhandled exception)
	AgentHealthProcessTerminated AgentHealthStatus = "ProcessTerminated"

	// AgentHealthStatusAgentFailure is when the opamp agent encountered an error during runtime.
	AgentFailure AgentHealthStatus = "AgentFailure"
)
