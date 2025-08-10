package status

const (

	// condition for the entire workload
	AgentInjectedStatus = "AgentInjected"

	// condition for a specific pod
	PodAgentInjectionStatus = "PodAgentInjection"
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


