package agent

type AgentHealthStatus string

const (
	HealthStatusHealthy                   AgentHealthStatus = "Healthy"
	HealthStatusUnknown                   AgentHealthStatus = "Unknown"
	HealthStatusStarting                  AgentHealthStatus = "Starting"
	HealthStatusUnsupportedRuntime        AgentHealthStatus = "UnsupportedRuntimeVersion"
	HealthStatusTerminated                AgentHealthStatus = "ProcessTerminated"
	HealthStatusAgentFailure              AgentHealthStatus = "AgentFailure"
	HealthStatusNoConnectionToOpAMPServer AgentHealthStatus = "NoConnectionToOpAMPServer"
)

func GetAgentHealthStatus(status string) AgentHealthStatus {
	switch status {
	case string(HealthStatusHealthy):
		return HealthStatusHealthy
	case string(HealthStatusStarting):
		return HealthStatusStarting
	case string(HealthStatusUnsupportedRuntime):
		return HealthStatusUnsupportedRuntime
	case string(HealthStatusNoConnectionToOpAMPServer):
		return HealthStatusNoConnectionToOpAMPServer
	case string(HealthStatusTerminated):
		return HealthStatusTerminated
	case string(HealthStatusAgentFailure):
		return HealthStatusAgentFailure
	}
	return HealthStatusUnknown
}
