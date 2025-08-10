package loaders

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	corev1 "k8s.io/api/core/v1"
)

type ComputedPodValues struct {
	AgentInjected       bool
	AgentInjectedStatus *model.DesiredConditionStatus
}

func NewComputedPodValues(pod *corev1.Pod, ic *v1alpha1.InstrumentationConfig) *ComputedPodValues {
	agentInjected, agentInjectedStatus := calculatePodAgentInjectedStatus(pod, ic)
	return &ComputedPodValues{
		AgentInjected:       agentInjected,
		AgentInjectedStatus: agentInjectedStatus,
	}
}

func calculatePodAgentInjectedStatus(pod *corev1.Pod, ic *v1alpha1.InstrumentationConfig) (bool, *model.DesiredConditionStatus) {
	agentHashValue, agentLabelExists := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]

	// if instrumentation config is missing, the agent should not be injected.
	if ic == nil {
		if !agentLabelExists {
			reasonStr := string(status.AgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentNotInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       status.PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "workload is not marked for instrumentation and agent is not injected as expected",
			}
		} else {
			reasonStr := string(status.AgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       status.PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressWaiting,
				ReasonEnum: &reasonStr,
				Message:    "workload is not marked for instrumentation and odigos agent is injected, this source is expected to rollout to replace with a new uninstrumented pod",
			}
		}
	}

	// at this point, we know the workload is is marked for instrumentation, since we have instrumentaiton config.

	// if the config sets agent injection to false, the agent should not be injected.
	if !ic.Spec.AgentInjectionEnabled {
		if !agentLabelExists {
			reasonStr := string(status.AgentInjectedReasonWorkloadAgnetDisabledAndNotInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       status.PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "agent is disabled for the source and agent is not injected as expected",
			}
		} else {
			reasonStr := string(status.AgentInjectedReasonWorkloadAgentDisabledButInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       status.PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressWaiting,
				ReasonEnum: &reasonStr,
				Message:    "agent is disabled for the source, but agent is injected, this kubernetesworkload is expected to rollout and replace this pod with an uninstrumented pod",
			}
		}
	}

	if agentLabelExists {
		sameHash := agentHashValue == ic.Spec.AgentsMetaHash
		if !sameHash {
			reasonStr := string(status.AgentInjectedReasonWorkloadAgentEnabledAndInjectedWithDifferentHash)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       status.PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressWaiting,
				ReasonEnum: &reasonStr,
				Message:    "source is enabled for agent injection but agent is injected with a different hash, this kubernetes workload is expected to rollout and replace this pod with an updated instrumented pod",
			}
		} else {
			reasonStr := string(status.AgentInjectedReasonWorkloadAgentEnabledAndInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       status.PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "source is enabled for agent injection and agent is injected as expected",
			}
		}
	}

	// no label for agent, and the workload is enabled.
	// check when it was instrumented.
	// TODO: record agent enabled time, the current value is when rollout completed.
	instrumentationTime := ic.Status.InstrumentationTime
	if instrumentationTime == nil {
		reasonStr := string(status.AgentInjectedReasonWorkloadAgentEnabledNotFinishRollout)
		return agentLabelExists, &model.DesiredConditionStatus{
			Name:       status.PodAgentInjectionStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    "source is enabled for agent injection but agent is not injected, this kubernetes workload is expected to rollout and replace this pod with an instrumented pod",
		}
	}

	podCreationTime := pod.CreationTimestamp
	if podCreationTime.Time.Before(instrumentationTime.Time) {
		reasonStr := string(status.AgentInjectedReasonWorkloadAgentEnabledAfterPodStarted)
		return agentLabelExists, &model.DesiredConditionStatus{
			Name:       status.PodAgentInjectionStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    "agent not injected because pod started before agent was enabled, expecting a rollout to terminated and replaced it with a new instrumented pod",
		}
	}

	reasonStr := string(status.AgentInjectedReasonWorkloadAgentEnabledAndNotInjected)
	return agentLabelExists, &model.DesiredConditionStatus{
		Name:       status.PodAgentInjectionStatus,
		Status:     model.DesiredStateProgressNotice,
		ReasonEnum: &reasonStr,
		Message:    "source is enabled for agent injection but agent is not injected, rollout the workload to replace this pod with a new instrumented pod",
	}
}
