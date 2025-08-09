package status

const (
	AgentInjectedStatus = "AgentInjected"
)

type AgentInjectedReason string

const (
	AgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentNotInjected AgentInjectedReason = "WorkloadNotMarkedForInstrumentationAgentNotInjected"
	AgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentInjected    AgentInjectedReason = "WorkloadNotMarkedForInstrumentationAgentInjected"
	AgentInjectedReasonWorkloadAgnetDisabledAndNotInjected                 AgentInjectedReason = "WorkloadAgnetDisabledAndNotInjected"
	AgentInjectedReasonWorkloadAgentDisabledButInjected                    AgentInjectedReason = "WorkloadAgentDisabledButInjected"
	AgentInjectedReasonWorkloadAgentEnabledAndInjected                     AgentInjectedReason = "WorkloadAgentEnabledAndInjected"
	AgentInjectedReasonWorkloadAgentEnabledAndNotInjected                  AgentInjectedReason = "WorkloadAgentEnabledAndNotInjected"
	AgentInjectedReasonWorkloadAgentEnabledAndInjectedWithDifferentHash    AgentInjectedReason = "WorkloadAgentEnabledAndInjectedWithDifferentHash"
	AgentInjectedReasonWorkloadAgentEnabledNotFinishRollout                AgentInjectedReason = "WorkloadAgentEnabledNotFinishRollout"
	AgentInjectedReasonWorkloadAgentEnabledAfterPodStarted                 AgentInjectedReason = "WorkloadAgentEnabledAfterPodStarted"
)
