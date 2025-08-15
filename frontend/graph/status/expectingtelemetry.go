package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	ExpectingTelemetryStatus = "ExpectingTelemetry"
)

type ExpectingTelemetryReason string

const (
	ExpectingTelemetryReasonWorkloadNotMarkedForInstrumentation ExpectingTelemetryReason = "WorkloadNotMarkedForInstrumentation"
	ExpectingTelemetryReasonAgentNotEnabledForInjection         ExpectingTelemetryReason = "AgentNotEnabledForInjection"
	ExpectingTelemetryReasonAgentNoRunningPod                   ExpectingTelemetryReason = "AgentNoRunningPod"
	ExpectingTelemetryReasonAgentNotInjected                    ExpectingTelemetryReason = "AgentNotInjected"
	ExpectingTelemetryReasonAgentInjectedButNoDataSent          ExpectingTelemetryReason = "AgentInjectedButNoDataSent"
	ExpectingTelemetryReasonAgentInjectedAndDataSent            ExpectingTelemetryReason = "AgentInjectedAndDataSent"
)

func CalculateExpectingTelemetryStatus(ic *v1alpha1.InstrumentationConfig, pods []computed.CachedPod, totalDataSentBytes *int) *model.K8sWorkloadTelemetryMetricsExpectingTelemetryStatus {
	expectingTelemetry := false

	// at the moment, a workload is expected to have telemetry
	// if the workload has agent injection enabled and at least one pod has the agent injected.
	if ic == nil {
		reasonStr := string(ExpectingTelemetryReasonWorkloadNotMarkedForInstrumentation)
		return &model.K8sWorkloadTelemetryMetricsExpectingTelemetryStatus{
			IsExpectingTelemetry: &expectingTelemetry,
			TelemetryObservedStatus: &model.DesiredConditionStatus{
				Name:       ExpectingTelemetryStatus,
				Status:     model.DesiredStateProgressIrrelevant,
				ReasonEnum: &reasonStr,
				Message:    "workload is not marked for instrumentation",
			},
		}
	}

	if !ic.Spec.AgentInjectionEnabled {
		reasonStr := string(ExpectingTelemetryReasonAgentNotEnabledForInjection)
		return &model.K8sWorkloadTelemetryMetricsExpectingTelemetryStatus{
			IsExpectingTelemetry: &expectingTelemetry,
			TelemetryObservedStatus: &model.DesiredConditionStatus{
				Name:       ExpectingTelemetryStatus,
				Status:     model.DesiredStateProgressIrrelevant,
				ReasonEnum: &reasonStr,
				Message:    "agent injection is not enabled for this workload",
			},
		}
	}

	if len(pods) == 0 {
		reasonStr := string(ExpectingTelemetryReasonAgentNoRunningPod)
		return &model.K8sWorkloadTelemetryMetricsExpectingTelemetryStatus{
			IsExpectingTelemetry: &expectingTelemetry,
			TelemetryObservedStatus: &model.DesiredConditionStatus{
				Name:       ExpectingTelemetryStatus,
				Status:     model.DesiredStateProgressPending,
				ReasonEnum: &reasonStr,
				Message:    "no running pods found for this workload",
			},
		}
	}

	for _, pod := range pods {
		if pod.AgentInjected {
			expectingTelemetry = true
			break
		}
	}

	if !expectingTelemetry {
		reasonStr := string(ExpectingTelemetryReasonAgentNotInjected)
		return &model.K8sWorkloadTelemetryMetricsExpectingTelemetryStatus{
			IsExpectingTelemetry: &expectingTelemetry,
			TelemetryObservedStatus: &model.DesiredConditionStatus{
				Name:       ExpectingTelemetryStatus,
				Status:     model.DesiredStateProgressIrrelevant,
				ReasonEnum: &reasonStr,
				Message:    "agent is not injected into any of the pods in this workload",
			},
		}
	}

	if totalDataSentBytes == nil || *totalDataSentBytes == 0 {
		reasonStr := string(ExpectingTelemetryReasonAgentInjectedButNoDataSent)
		return &model.K8sWorkloadTelemetryMetricsExpectingTelemetryStatus{
			IsExpectingTelemetry: &expectingTelemetry,
			TelemetryObservedStatus: &model.DesiredConditionStatus{
				Name:       ExpectingTelemetryStatus,
				Status:     model.DesiredStateProgressWaiting,
				ReasonEnum: &reasonStr,
				Message:    "agent is injected into the pods in this workload, but no telemetry data was sent yet",
			},
		}
	}

	reasonStr := string(ExpectingTelemetryReasonAgentInjectedAndDataSent)
	return &model.K8sWorkloadTelemetryMetricsExpectingTelemetryStatus{
		IsExpectingTelemetry: &expectingTelemetry,
		TelemetryObservedStatus: &model.DesiredConditionStatus{
			Name:       ExpectingTelemetryStatus,
			Status:     model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr,
			Message:    "agent is injected into the pods in this workload, and telemetry data was sent",
		},
	}
}
