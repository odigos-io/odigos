package status

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	corev1 "k8s.io/api/core/v1"
)

const (

	// condition for the entire workload
	AgentInjectedStatus = "AgentInjected"

	// condition for a specific pod
	PodAgentInjectionStatus = "PodAgentInjection"
)

type PodAgentInjectedReason string

const (
	PodAgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentNotInjected PodAgentInjectedReason = "WorkloadNotMarkedForInstrumentationAgentNotInjected"
	PodAgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentInjected    PodAgentInjectedReason = "WorkloadNotMarkedForInstrumentationAgentInjected"
	PodAgentInjectedReasonWorkloadAgnetDisabledAndNotInjected                 PodAgentInjectedReason = "WorkloadAgnetDisabledAndNotInjected"
	PodAgentInjectedReasonWorkloadAgentDisabledButInjected                    PodAgentInjectedReason = "WorkloadAgentDisabledButInjected"
	PodAgentInjectedReasonWorkloadAgentEnabledAndInjected                     PodAgentInjectedReason = "WorkloadAgentEnabledAndInjected"
	PodAgentInjectedReasonWorkloadAgentEnabledAndNotInjected                  PodAgentInjectedReason = "WorkloadAgentEnabledAndNotInjected"
	PodAgentInjectedReasonWorkloadAgentEnabledAndInjectedWithDifferentHash    PodAgentInjectedReason = "WorkloadAgentEnabledAndInjectedWithDifferentHash"
	PodAgentInjectedReasonWorkloadAgentEnabledNotFinishRollout                PodAgentInjectedReason = "WorkloadAgentEnabledNotFinishRollout"
	PodAgentInjectedReasonWorkloadAgentEnabledAfterPodStarted                 PodAgentInjectedReason = "WorkloadAgentEnabledAfterPodStarted"
)

type AgentInjectionReason string

const (
	AgentInjectionReasonNoRunningPods            AgentInjectionReason = "NoRunningPods"
	AgentInjectionReasonAllPodsAgentInjected     AgentInjectionReason = "AllPodsAgentInjected"
	AgentInjectionReasonAllPodsAgentNotInjected  AgentInjectionReason = "AllPodsAgentNotInjected"
	AgentInjectionReasonSomePodsAgentNotInjected AgentInjectionReason = "SomePodsAgentNotInjected"
	AgentInjectionReasonSomePodsAgentInjected    AgentInjectionReason = "SomePodsAgentInjected"
)

func CalculatePodAgentInjectedStatus(pod *corev1.Pod, ic *v1alpha1.InstrumentationConfig) (bool, *model.DesiredConditionStatus) {
	agentHashValue, agentLabelExists := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]

	// if instrumentation config is missing, the agent should not be injected.
	if ic == nil {
		if !agentLabelExists {
			reasonStr := string(PodAgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentNotInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "workload is not marked for instrumentation and agent is not injected as expected",
			}
		} else {
			reasonStr := string(PodAgentInjectedReasonWorkloadNotMarkedForInstrumentationAgentInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
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
			reasonStr := string(PodAgentInjectedReasonWorkloadAgnetDisabledAndNotInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "agent is disabled for the source and agent is not injected as expected",
			}
		} else {
			reasonStr := string(PodAgentInjectedReasonWorkloadAgentDisabledButInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressWaiting,
				ReasonEnum: &reasonStr,
				Message:    "agent is disabled for the source, but agent is injected, this kubernetesworkload is expected to rollout and replace this pod with an uninstrumented pod",
			}
		}
	}

	if agentLabelExists {
		sameHash := agentHashValue == ic.Spec.AgentsMetaHash
		if !sameHash {
			reasonStr := string(PodAgentInjectedReasonWorkloadAgentEnabledAndInjectedWithDifferentHash)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressWaiting,
				ReasonEnum: &reasonStr,
				Message:    "source is enabled for agent injection but agent is injected with a different hash, this kubernetes workload is expected to rollout and replace this pod with an updated instrumented pod",
			}
		} else {
			reasonStr := string(PodAgentInjectedReasonWorkloadAgentEnabledAndInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
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
		reasonStr := string(PodAgentInjectedReasonWorkloadAgentEnabledNotFinishRollout)
		return agentLabelExists, &model.DesiredConditionStatus{
			Name:       PodAgentInjectionStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    "source is enabled for agent injection but agent is not injected, this kubernetes workload is expected to rollout and replace this pod with an instrumented pod",
		}
	}

	podCreationTime := pod.CreationTimestamp
	if podCreationTime.Time.Before(instrumentationTime.Time) {
		reasonStr := string(PodAgentInjectedReasonWorkloadAgentEnabledAfterPodStarted)
		return agentLabelExists, &model.DesiredConditionStatus{
			Name:       PodAgentInjectionStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    "agent not injected because pod started before agent was enabled, expecting a rollout to terminated and replaced it with a new instrumented pod",
		}
	}

	reasonStr := string(PodAgentInjectedReasonWorkloadAgentEnabledAndNotInjected)
	return agentLabelExists, &model.DesiredConditionStatus{
		Name:       PodAgentInjectionStatus,
		Status:     model.DesiredStateProgressNotice,
		ReasonEnum: &reasonStr,
		Message:    "source is enabled for agent injection but agent is not injected, rollout the workload to replace this pod with a new instrumented pod",
	}
}

func CalculateAgentInjectedStatus(ic *v1alpha1.InstrumentationConfig, pods []computed.CachedPod) *model.DesiredConditionStatus {
	if len(pods) == 0 {
		reasonStr := string(AgentInjectionReasonNoRunningPods)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressIrrelevant,
			ReasonEnum: &reasonStr,
			Message:    "no pods found for this workload",
		}
	}

	numSuccess := 0
	numNotSuccess := 0
	for _, pod := range pods {
		if pod.AgentInjectedStatus.Status == model.DesiredStateProgressSuccess {
			numSuccess++
		} else {
			numNotSuccess++
		}
	}

	// if ic is nil, we assume agent is not enabled.
	agentEnabled := false
	if ic != nil {
		agentEnabled = ic.Spec.AgentInjectionEnabled
	}

	if numNotSuccess > 0 {
		var reasonStr, message string
		if agentEnabled {
			reasonStr = string(AgentInjectionReasonSomePodsAgentNotInjected)
			message = fmt.Sprintf("%d/%d pods should have agent injected, but do not", numNotSuccess, len(pods))
		} else {
			reasonStr = string(AgentInjectionReasonSomePodsAgentInjected)
			message = fmt.Sprintf("%d/%d pods have agent injected when it should not", numNotSuccess, len(pods))
		}
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    message,
		}
	} else {
		var reasonStr, message string
		if agentEnabled {
			reasonStr = string(AgentInjectionReasonAllPodsAgentInjected)
			message = fmt.Sprintf("all %d pods have odigos agent injected as expected", numSuccess)
		} else {
			reasonStr = string(AgentInjectionReasonAllPodsAgentNotInjected)
			message = fmt.Sprintf("all %d pods do not have odigos agent injected as expected", numSuccess)
		}
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr,
			Message:    message,
		}
	}
}
