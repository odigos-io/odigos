package status

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

const (
	PodContainerHealthOdigosStatus = "PodContainerHealthOdigos"
	PodHealthOdigosStatus          = "PodHealthOdigos"
)

type PodHealthOdigosStatusReason string

const (
	PodHealthOdigosStatusReasonHealthy                        PodHealthOdigosStatusReason = "Healthy"
	PodHealthOdigosStatusReasonNotInjected                    PodHealthOdigosStatusReason = "OdigosAgentNotInjected"
	PodHealthOdigosStatusReasonInjectedUninstrumentedSource   PodHealthOdigosStatusReason = "OdigosAgentInjectedInUninstrumentedSource"
	PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy PodHealthOdigosStatusReason = "InstrumentatedProcessUnhealthy"
	PodHealthOdigosStatusReasonNoInstrumentedProcesses        PodHealthOdigosStatusReason = "NoInstrumentedProcesses"
	PodHealthOdigosStatusReasonNoPods                         PodHealthOdigosStatusReason = "NoPods"
)

func createPodContainerHealthOdigosStatus(reason PodHealthOdigosStatusReason, message string, status model.DesiredStateProgress) *model.DesiredConditionStatus {
	reasonStr := string(reason)
	return &model.DesiredConditionStatus{
		Name:       PodContainerHealthOdigosStatus,
		Status:     status,
		ReasonEnum: &reasonStr,
		Message:    message,
	}
}

func createPodHealthOdigosStatus(reason PodHealthOdigosStatusReason, message string, status model.DesiredStateProgress) *model.DesiredConditionStatus {
	reasonStr := string(reason)
	return &model.DesiredConditionStatus{
		Name:       PodHealthOdigosStatus,
		Status:     status,
		ReasonEnum: &reasonStr,
		Message:    message,
	}
}

func CalculatePodContainerHealthOdigosStatus(container *computed.ComputedPodContainer, containerConfig *v1alpha1.ContainerAgentConfig, iis []*v1alpha1.InstrumentationInstance) *model.DesiredConditionStatus {

	// non agent (either as direct injection or with "no restart" injection)
	if container.OtelDistroName == nil && (containerConfig == nil || !containerConfig.PodManifestInjectionOptional) {
		return nil
	}

	// first check if any process is unhealthy
	for _, ii := range iis {
		if ii.Status.Healthy != nil && !*ii.Status.Healthy {
			return createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy, "instrumented process in the container is unhealthy", model.DesiredStateProgressFailure)
		}
		for _, c := range ii.Status.Components {
			if c.Healthy != nil && !*c.Healthy {
				return createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy, "unhealthy instrumentation library in instrumented process", model.DesiredStateProgressFailure)
			}
		}
	}

	if container.ExpectingInstrumentationInstances && len(iis) == 0 {
		return createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonNoInstrumentedProcesses, "no instrumented processes found in the container", model.DesiredStateProgressWaiting)
	}

	var healthyMessage string
	if len(iis) > 0 {
		healthyMessage = "all instrumented processes are reported as healthy"
	} else {
		healthyMessage = "odigos agent is running in the container"
	}

	return createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonHealthy, healthyMessage, model.DesiredStateProgressSuccess)
}

func CalculatePodHealthOdigosStatus(computedPod *computed.CachedPod, containersOdigosHealthConditions []*model.DesiredConditionStatus) *model.DesiredConditionStatus {

	// if agent injected is not successful, the odigos is not healthy for this pod.
	if computedPod.AgentInjectedStatus.Status != model.DesiredStateProgressSuccess {
		var reason PodHealthOdigosStatusReason
		if computedPod.AgentInjected {
			reason = PodHealthOdigosStatusReasonInjectedUninstrumentedSource
		} else {
			reason = PodHealthOdigosStatusReasonNotInjected
		}
		return createPodHealthOdigosStatus(reason, computedPod.AgentInjectedStatus.Message, computedPod.AgentInjectedStatus.Status)
	}

	containerStatues := AggregateConditionsBySeverity(containersOdigosHealthConditions)
	if containerStatues == nil {
		return nil
	}

	if containerStatues.Status != model.DesiredStateProgressSuccess {
		return containerStatues
	}

	return createPodHealthOdigosStatus(PodHealthOdigosStatusReasonHealthy, "odigos instrumentation in pod is healthy", model.DesiredStateProgressSuccess)
}

func MostSeverPodStatusToAggregated(mostSeverPodStatus *model.DesiredConditionStatus) *model.DesiredConditionStatus {
	if mostSeverPodStatus == nil || mostSeverPodStatus.ReasonEnum == nil {
		return nil
	}

	switch *mostSeverPodStatus.ReasonEnum {
	case string(PodHealthOdigosStatusReasonHealthy):
		return createPodHealthOdigosStatus(PodHealthOdigosStatusReasonHealthy, "odigos is healthy in all pods", model.DesiredStateProgressSuccess)
	case string(PodHealthOdigosStatusReasonNotInjected):
		return createPodHealthOdigosStatus(PodHealthOdigosStatusReasonNotInjected, "not all pods are running with odigos agent", mostSeverPodStatus.Status)
	case string(PodHealthOdigosStatusReasonInjectedUninstrumentedSource):
		return createPodHealthOdigosStatus(PodHealthOdigosStatusReasonInjectedUninstrumentedSource, "not all pods are running without odigos agent", mostSeverPodStatus.Status)
	case string(PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy),
		string(PodHealthOdigosStatusReasonNoInstrumentedProcesses),
		string(PodHealthOdigosStatusReasonNoPods):
		return mostSeverPodStatus
	default:
		return mostSeverPodStatus
	}

}
