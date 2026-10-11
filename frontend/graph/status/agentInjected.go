package status

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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

	// not marked for instrumentation
	PodAgentInjectedReasonNotMarkedNotInjected   PodAgentInjectedReason = "NotMarkedNotInjected"
	PodAgentInjectedReasonNotMarkedAutoRollout   PodAgentInjectedReason = "NotMarkedAutoRollout"
	PodAgentInjectedReasonNotMarkedManualRollout PodAgentInjectedReason = "NotMarkedManualRollout"

	// disabled for agent injection
	PodAgentInjectedReasonDisabledNotInjected   PodAgentInjectedReason = "DisabledNotInjected"
	PodAgentInjectedReasonDisabledAutoRollout   PodAgentInjectedReason = "DisabledAutoRollout"
	PodAgentInjectedReasonDisabledManualRollout PodAgentInjectedReason = "DisabledManualRollout"

	// pod manifest injection optional (no restart required to enable agent injection)
	PodAgentInjectedReasonPodManifestInjectionOptional PodAgentInjectedReason = "PodManifestInjectionOptional"

	// out of date
	PodAgentInjectedReasonOutOfDateAutoRollout   PodAgentInjectedReason = "OutOfDateAutoRollout"
	PodAgentInjectedReasonOutOfDateManualRollout PodAgentInjectedReason = "OutOfDateManualRollout"

	// old pods
	PodAgentInjectedReasonEnabledAfterPodCreatedAutoRollout   PodAgentInjectedReason = "EnabledAfterPodCreatedAutoRollout"
	PodAgentInjectedReasonEnabledAfterPodCreatedManualRollout PodAgentInjectedReason = "EnabledAfterPodCreatedManualRollout"

	// others
	PodAgentInjectedReasonSuccessfullyInjected PodAgentInjectedReason = "SuccessfullyInjected"
	PodAgentInjectedReasonEnabledNotInjected   PodAgentInjectedReason = "EnabledNotInjected"
)

type AgentInjectionReason string

const (
	AgentInjectionReasonNoRunningPods AgentInjectionReason = "NoRunningPods"

	// this means that agent should not be injected and this is the status
	AgentInjecteonReasonNotInjectedAsExpected AgentInjectionReason = "AgentNotInjectedAsExpected"

	// some pods still have agent injected when they should not, waiting for automatic rollout to replace them.
	AgentInjectionReasonSomePodsAgentInjectedWaitingForAutoRollout AgentInjectionReason = "SomePodsAgentInjectedWaitingForAutoRollout"

	// some pods still have agent injected when they should not, waiting for manual rollout to replace them.
	AgentInjectionReasonSomePodsAgentInjectedRolloutNeeded AgentInjectionReason = "SomePodsAgentInjectedRolloutNeeded"

	// all pods should have agent injected and they do.
	AgentInjectionReasonAllPodsAgentInjected AgentInjectionReason = "AgentInjectedAsExpected"

	// some old pods still have agent not injected when they should, waiting for automatic rollout to replace them.
	AgentInjectionReasonSomePodsAgentNotInjectedWaitingForAutoRollout AgentInjectionReason = "SomePodsAgentNotInjectedWaitingForAutoRollout"

	// some old pods still have agent not injected when they should, waiting for manual rollout to replace them.
	AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded AgentInjectionReason = "SomePodsAgentNotInjectedRolloutNeeded"

	// some pods are not running up to date version of the agent, waiting for automatic rollout to replace them.
	AgentInjectionReasonSomePodsAgentOutOfDateWaitingForAutoRollout AgentInjectionReason = "SomePodsAgentOutOfDateWaitingForAutoRollout"

	// some pods are not running up to date version of the agent, waiting for manual rollout to replace them.
	AgentInjectionReasonSomePodsAgentOutOfDateRolloutNeeded AgentInjectionReason = "SomePodsAgentOutOfDateRolloutNeeded"

	// some pods that started after agent was enabled have no agent injected.
	// probably instrumentor webhook failed to run for some reason.
	AgentInjectionReasonSomePodsAgentNotInjectedWebhookMissed AgentInjectionReason = "SomePodsAgentNotInjectedWebhookMissed"
)

func CalculatePodAgentInjectedStatus(pod *corev1.Pod, ic *v1alpha1.InstrumentationConfig, automaticRolloutEnabled bool) (bool, *model.DesiredConditionStatus) {
	agentHashValue, agentLabelExists := pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel]

	// if instrumentation config is missing, the agent should not be injected.
	if ic == nil {
		if !agentLabelExists {
			reasonStr := string(PodAgentInjectedReasonNotMarkedNotInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "source is not marked for instrumentation; odigos agent is not injected to pod (expected)",
			}
		} else {
			// diffrentiate between automatic rollout enabled and disabled.
			if automaticRolloutEnabled {
				reasonStr := string(PodAgentInjectedReasonNotMarkedAutoRollout)
				return agentLabelExists, &model.DesiredConditionStatus{
					Name:       PodAgentInjectionStatus,
					Status:     model.DesiredStateProgressWaiting,
					ReasonEnum: &reasonStr,
					Message:    "source is not marked for instrumentation and odigos agent is injected; this source will be rolled out automatically by odigos to replace with new uninstrumented pods",
				}
			} else {
				reasonStr := string(PodAgentInjectedReasonNotMarkedManualRollout)
				return agentLabelExists, &model.DesiredConditionStatus{
					Name:       PodAgentInjectionStatus,
					Status:     model.DesiredStateProgressNotice,
					ReasonEnum: &reasonStr,
					Message:    "source is not marked for instrumentation and odigos agent is injected; rollout this source to start new uninstrumented pods",
				}
			}
		}
	}

	// at this point, we know the workload is marked for instrumentation, since we have instrumentaiton config.
	pw, _ := workload.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name, ic.Namespace)
	workloadKind := string(pw.Kind)

	// if the config sets agent injection enabled to false, the agent should not be injected.
	// for example: ignored containers, unsupported programming language, etc.
	if !ic.Spec.AgentInjectionEnabled {
		if !agentLabelExists {
			reasonStr := string(PodAgentInjectedReasonDisabledNotInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "source is disabled for agent injection; odigos agent is not injected (expected)",
			}
		} else {
			// diffrentiate between automatic rollout enabled and disabled.
			if automaticRolloutEnabled {
				reasonStr := string(PodAgentInjectedReasonDisabledAutoRollout)
				return agentLabelExists, &model.DesiredConditionStatus{
					Name:       PodAgentInjectionStatus,
					Status:     model.DesiredStateProgressWaiting,
					ReasonEnum: &reasonStr,
					Message:    fmt.Sprintf("%s is disabled for agent injection but odigos agent is injected; this %s will be rolled out automatically by odigos", workloadKind, workloadKind),
				}
			} else {
				reasonStr := string(PodAgentInjectedReasonDisabledManualRollout)
				return agentLabelExists, &model.DesiredConditionStatus{
					Name:       PodAgentInjectionStatus,
					Status:     model.DesiredStateProgressNotice, // action item - restart this source.
					ReasonEnum: &reasonStr,
					Message:    fmt.Sprintf("%s is disabled for agent injection but odigos agent is injected; rollout this %s to replace with new uninstrumented pods", workloadKind, workloadKind),
				}
			}
		}
	}

	// if the pod manifest injection is optional, we want to show this as success,
	// since the agent can be enabled without a restart
	podManifestInjectionOptional := ic.Spec.PodManifestInjectionOptional

	if agentLabelExists {
		sameHash := agentHashValue == ic.Spec.AgentsMetaHash
		if sameHash {
			// this is the common happy path. both the source and the agent are marked for instrumentation and the hash is the same.
			reasonStr := string(PodAgentInjectedReasonSuccessfullyInjected)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressSuccess,
				ReasonEnum: &reasonStr,
				Message:    "odigos agent is successfully injected to this pod",
			}
		} else {
			if podManifestInjectionOptional {
				reasonStr := string(PodAgentInjectedReasonPodManifestInjectionOptional)
				return agentLabelExists, &model.DesiredConditionStatus{
					Name:       PodAgentInjectionStatus,
					Status:     model.DesiredStateProgressSuccess,
					ReasonEnum: &reasonStr,
					Message:    "this agent is automatically enabled in running pod",
				}
			}

			// this is the rare case where the source and the agent are marked for instrumentation but the hash is different.
			// it can happen when migrating from OSS <-> Enterprise, or when agent version is updated in a way that requires restarts.
			// pods need to restart for new pods to have the correct agent hash.
			if automaticRolloutEnabled {
				reasonStr := string(PodAgentInjectedReasonOutOfDateAutoRollout)
				return agentLabelExists, &model.DesiredConditionStatus{
					Name:       PodAgentInjectionStatus,
					Status:     model.DesiredStateProgressWaiting,
					ReasonEnum: &reasonStr,
					Message:    fmt.Sprintf("odigos agent is not up to date; this %s will be rolled out automatically by odigos", workloadKind),
				}
			} else {
				reasonStr := string(PodAgentInjectedReasonOutOfDateManualRollout)
				return agentLabelExists, &model.DesiredConditionStatus{
					Name:       PodAgentInjectionStatus,
					Status:     model.DesiredStateProgressNotice,
					ReasonEnum: &reasonStr,
					Message:    fmt.Sprintf("odigos agent is not up to date; rollout this %s to start new pods with latest agent version", workloadKind),
				}
			}
		}
	}

	// if the pod manifest injection optional, it's ok, agent is enabled regardless of the hash or pod manifest changes.
	if podManifestInjectionOptional {
		reasonStr := string(PodAgentInjectedReasonPodManifestInjectionOptional)
		return agentLabelExists, &model.DesiredConditionStatus{
			Name:       PodAgentInjectionStatus,
			Status:     model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr,
			Message:    "this agent is automatically enabled in running pod",
		}
	}

	// at this point:
	// - the source is marked for instrumentation
	// - agent injection is enabled
	// - the pod has no odigos label (agent is not injected)
	// - the pod manifest injection is required (e.g. an agent in at least one container requires pod restart to be enabled)
	//
	// there can be few options here:
	// 1. automatic rollout is awaiting or in progress
	// 2. manual rollout is needed
	// 3. instrumentor webhook failed to inject the agent
	//
	// these are being differentiated by the time at which the agent meta hash changed.
	// all pods after this time are expected to have the label with the correct agent hash.
	instrumentationTime := ic.Spec.AgentsMetaHashChangedTime
	if instrumentationTime == nil { // support for sources that were created before this field was added.
		reasonStr := string(PodAgentInjectedReasonEnabledNotInjected)
		return agentLabelExists, &model.DesiredConditionStatus{
			Name:       PodAgentInjectionStatus,
			Status:     model.DesiredStateProgressNotice,
			ReasonEnum: &reasonStr,
			Message:    "source is enabled for agent injection but odigos agent was not injected; rollout the workload to replace this pod with an instrumented one",
		}
	}

	podCreationTime := pod.CreationTimestamp
	// pod created before agent was enabled (not up to date)
	if podCreationTime.Time.Before(instrumentationTime.Time) {
		if automaticRolloutEnabled {
			reasonStr := string(PodAgentInjectedReasonEnabledAfterPodCreatedAutoRollout)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressWaiting,
				ReasonEnum: &reasonStr,
				Message:    "old pod - created before agent was enabled; will be rolled out automatically by odigos",
			}
		} else {
			reasonStr := string(PodAgentInjectedReasonEnabledAfterPodCreatedManualRollout)
			return agentLabelExists, &model.DesiredConditionStatus{
				Name:       PodAgentInjectionStatus,
				Status:     model.DesiredStateProgressNotice,
				ReasonEnum: &reasonStr,
				Message:    "old pod - created before agent was enabled; rollout the workload to replace this pod with an instrumented one",
			}
		}
	}

	// at this point:
	// - the source is marked for instrumentation
	// - agent injection is enabled
	// - the pod has no odigos label (agent is not injected)
	// - the pod was created after agent was enabled
	//
	// this means that:
	// 1. instrumentor webhook failed to run (instrumentor down)
	// 2. instrumentor webhook returned an error which failed the injection
	// 3. pods created right after the timestamp was taken and before webhook synced on the change.
	reasonStr := string(PodAgentInjectedReasonEnabledNotInjected)
	return agentLabelExists, &model.DesiredConditionStatus{
		Name:       PodAgentInjectionStatus,
		Status:     model.DesiredStateProgressNotice,
		ReasonEnum: &reasonStr,
		Message:    "agent is not injected to this pod; check for instrumentor component health and rollout the workload to replace this pod with an instrumented one",
	}
}

func CalculateAgentInjectedStatus(ic *v1alpha1.InstrumentationConfig, pods []computed.CachedPod) *model.DesiredConditionStatus {

	if len(pods) == 0 {
		reasonStr := string(AgentInjectionReasonNoRunningPods)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressIrrelevant,
			ReasonEnum: &reasonStr,
			Message:    "no pods found for this source",
		}
	}

	// aggregate the reasons per different pods, to show one status for all of them.
	reasonsMap := map[PodAgentInjectedReason]int{}
	for _, pod := range pods {
		if pod.AgentInjectedStatus != nil && pod.AgentInjectedStatus.ReasonEnum != nil {
			reasonsMap[PodAgentInjectedReason(*pod.AgentInjectedStatus.ReasonEnum)]++
		}
	}

	// check the reasons based on severity, and return the most important reason.

	// ======= Not marked for instrumentation =======
	if num, found := reasonsMap[PodAgentInjectedReasonNotMarkedManualRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentInjectedRolloutNeeded)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressNotice,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("source not marked for instrumentation but %d/%d pods are running odigos agent; rollout this source to replace these pods with uninstrumented ones", num, len(pods)),
		}
	}
	if num, found := reasonsMap[PodAgentInjectedReasonNotMarkedAutoRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentInjectedWaitingForAutoRollout)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("odigos agent is injected in %d/%d pods; odigos will roll out these pods automatically", num, len(pods)),
		}
	}
	if _, found := reasonsMap[PodAgentInjectedReasonNotMarkedNotInjected]; found {
		reasonStr := string(AgentInjecteonReasonNotInjectedAsExpected)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr,
			Message:    "odigos agent is not injected as expected since source is not marked for instrumentation",
		}
	}

	// ======= Disabled for agent injection =======
	if num, found := reasonsMap[PodAgentInjectedReasonDisabledAutoRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentInjectedWaitingForAutoRollout)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("source is disabled for agent injection but %d/%d pods are running odigos agent; odigos will roll out these pods automatically", num, len(pods)),
		}
	}
	if num, found := reasonsMap[PodAgentInjectedReasonDisabledManualRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentInjectedRolloutNeeded)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressNotice,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("source is disabled for agent injection but %d/%d pods are running odigos agent; rollout this source to replace these pods with uninstrumented ones", num, len(pods)),
		}
	}
	if num, found := reasonsMap[PodAgentInjectedReasonDisabledNotInjected]; found {
		reasonStr := string(AgentInjecteonReasonNotInjectedAsExpected)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("source is disabled for agent injection but %d/%d pods are running odigos agent; rollout this source to replace these pods with uninstrumented ones", num, len(pods)),
		}
	}

	// ======= Pod manifest injection optional =======
	if _, found := reasonsMap[PodAgentInjectedReasonPodManifestInjectionOptional]; found {
		reasonStr := string(PodAgentInjectedReasonPodManifestInjectionOptional)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr,
			Message:    "this agent is automatically enabled in running pod",
		}
	}

	// ======= Enabled after pod created =======
	if num, found := reasonsMap[PodAgentInjectedReasonEnabledNotInjected]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressNotice,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("%d/%d pods are running without odigos agent and require restart to apply instrumentation; check instrumentor component health and trigger a rollout", num, len(pods)),
		}
	}
	if num, found := reasonsMap[PodAgentInjectedReasonEnabledAfterPodCreatedManualRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressNotice,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("%d/%d pods are running without odigos agent and require restart to apply instrumentation; trigger a rollout to replace these pods with instrumented ones", num, len(pods)),
		}
	}
	if num, found := reasonsMap[PodAgentInjectedReasonEnabledAfterPodCreatedAutoRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("%d/%d pods are running without odigos agent and require restart to apply instrumentation; trigger a rollout to replace these pods with instrumented ones", num, len(pods)),
		}
	}

	// ======= Out of date auto rollout =======
	if num, found := reasonsMap[PodAgentInjectedReasonOutOfDateManualRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentOutOfDateRolloutNeeded)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressNotice,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("%d/%d pods are running without odigos agent and require restart to apply instrumentation; trigger a rollout to replace these pods with instrumented ones", num, len(pods)),
		}
	}
	if num, found := reasonsMap[PodAgentInjectedReasonOutOfDateAutoRollout]; found {
		reasonStr := string(AgentInjectionReasonSomePodsAgentOutOfDateWaitingForAutoRollout)
		return &model.DesiredConditionStatus{
			Name:       AgentInjectedStatus,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("%d/%d pods are running without odigos agent and require restart to apply instrumentation; odigos will roll out these pods automatically", num, len(pods)),
		}
	}

	reasonStr := string(PodAgentInjectedReasonSuccessfullyInjected)
	return &model.DesiredConditionStatus{
		Name:       AgentInjectedStatus,
		Status:     model.DesiredStateProgressSuccess,
		ReasonEnum: &reasonStr,
		Message:    fmt.Sprintf("all %d pods are instrumented as expected", len(pods)),
	}
}
