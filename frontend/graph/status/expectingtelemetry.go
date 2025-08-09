package status

const (
	ExpectingTelemetryStatus = "ExpectingTelemetry"
)

type ExpectingTelemetryReason string

const (
	ExpectingTelemetryReasonAgentNotEnabledForInjection AgentInjectedReason = "AgentNotEnabledForInjection"
	ExpectingTelemetryReasonAgentNoRunningPod           AgentInjectedReason = "AgentNoRunningPod"
	ExpectingTelemetryReasonAgentNotInjected            AgentInjectedReason = "AgentNotInjected"
	ExpectingTelemetryReasonAgentInjectedButNoDataSent  AgentInjectedReason = "AgentInjectedButNoDataSent"
	ExpectingTelemetryReasonAgentInjectedAndDataSent    AgentInjectedReason = "AgentInjectedAndDataSent"
)
