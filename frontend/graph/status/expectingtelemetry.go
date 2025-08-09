package status

const (
	ExpectingTelemetryStatus = "ExpectingTelemetry"
)

type ExpectingTelemetryReason string

const (
	ExpectingTelemetryReasonWorkloadNotMarkedForInstrumentation AgentInjectedReason = "WorkloadNotMarkedForInstrumentation"
	ExpectingTelemetryReasonAgentNotEnabledForInjection         AgentInjectedReason = "AgentNotEnabledForInjection"
	ExpectingTelemetryReasonAgentNoRunningPod                   AgentInjectedReason = "AgentNoRunningPod"
	ExpectingTelemetryReasonAgentNotInjected                    AgentInjectedReason = "AgentNotInjected"
	ExpectingTelemetryReasonAgentInjectedButNoDataSent          AgentInjectedReason = "AgentInjectedButNoDataSent"
	ExpectingTelemetryReasonAgentInjectedAndDataSent            AgentInjectedReason = "AgentInjectedAndDataSent"
)
