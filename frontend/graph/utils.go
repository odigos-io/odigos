package graph

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
)

func emptyStrToNil(str string) *string {
	if str == "" {
		return nil
	}
	return &str
}

func distroParamsToModel(distroParams map[string]string) []*model.DistroParam {
	modelDistroParams := make([]*model.DistroParam, 0, len(distroParams))
	for name, value := range distroParams {
		modelDistroParams = append(modelDistroParams, &model.DistroParam{
			Name:  name,
			Value: value,
		})
	}
	return modelDistroParams
}

func envVarsToModel(envVars []v1alpha1.EnvVar) []*model.EnvVar {
	modelEnvVars := make([]*model.EnvVar, len(envVars))
	for i, envVar := range envVars {
		modelEnvVars[i] = &model.EnvVar{
			Name:  envVar.Name,
			Value: envVar.Value,
		}
	}
	return modelEnvVars
}

func runtimeDetailsContainersToModel(runtimeDetails *v1alpha1.RuntimeDetailsByContainer) *model.K8sWorkloadRuntimeInfoContainer {
	containerName := runtimeDetails.ContainerName

	var runtimeVersion *string
	if runtimeDetails.RuntimeVersion != "" {
		runtimeVersion = &runtimeDetails.RuntimeVersion
	}
	var otherAgentName *string
	if runtimeDetails.OtherAgent != nil {
		otherAgentName = &runtimeDetails.OtherAgent.Name
	}
	var libcType *string
	if runtimeDetails.LibCType != nil {
		libcTypeStr := string(*runtimeDetails.LibCType)
		libcType = &libcTypeStr
	}
	return &model.K8sWorkloadRuntimeInfoContainer{
		ContainerName:           containerName,
		Language:                model.ProgrammingLanguage(runtimeDetails.Language),
		RuntimeVersion:          runtimeVersion,
		ProcessEnvVars:          envVarsToModel(runtimeDetails.EnvVars),
		ContainerRuntimeEnvVars: envVarsToModel(runtimeDetails.EnvFromContainerRuntime),
		CriErrorMessage:         runtimeDetails.CriErrorMessage,
		LibcType:                libcType,
		SecureExecutionMode:     runtimeDetails.SecureExecutionMode,
		OtherAgentName:          otherAgentName,
	}
}

func agentEnabledContainersToModel(containerAgentConfig *v1alpha1.ContainerAgentConfig) *model.K8sWorkloadAgentEnabledContainer {
	var envInjectionMethodStr *string
	if containerAgentConfig.EnvInjectionMethod != nil {
		asStr := string(*containerAgentConfig.EnvInjectionMethod)
		envInjectionMethodStr = &asStr
	}

	var traces *model.K8sWorkloadAgentEnabledContainerTraces
	if containerAgentConfig.Traces != nil {
		traces = &model.K8sWorkloadAgentEnabledContainerTraces{
			Enabled: true,
		}
	}
	var metrics *model.K8sWorkloadAgentEnabledContainerMetrics
	if containerAgentConfig.Metrics != nil {
		metrics = &model.K8sWorkloadAgentEnabledContainerMetrics{
			Enabled: true,
		}
	}
	var logs *model.K8sWorkloadAgentEnabledContainerLogs
	if containerAgentConfig.Logs != nil {
		logs = &model.K8sWorkloadAgentEnabledContainerLogs{
			Enabled: true,
		}
	}

	return &model.K8sWorkloadAgentEnabledContainer{
		ContainerName:      containerAgentConfig.ContainerName,
		AgentEnabled:       containerAgentConfig.AgentEnabled,
		AgentEnabledStatus: status.CalculateAgentInjectionEnabledStatusForContainer(containerAgentConfig),
		OtelDistroName:     emptyStrToNil(containerAgentConfig.OtelDistroName),
		EnvInjectionMethod: envInjectionMethodStr,
		DistroParams:       distroParamsToModel(containerAgentConfig.DistroParams),
		Traces:             traces,
		Metrics:            metrics,
		Logs:               logs,
	}
}

// givin a desired state progress enum, return a value to determine the order of severity.
// the lower the number, the more sever the state is
func desiredStateProgressSeverity(desiredStateProgress model.DesiredStateProgress) int {
	switch desiredStateProgress {
	case model.DesiredStateProgressError:
		return 0
	case model.DesiredStateProgressFailure:
		return 10
	case model.DesiredStateProgressNotice:
		return 20
	case model.DesiredStateProgressPending:
		return 30
	case model.DesiredStateProgressWaiting:
		return 40
	case model.DesiredStateProgressUnsupported:
		return 50
	case model.DesiredStateProgressDisabled:
		return 60
	case model.DesiredStateProgressSuccess:
		return 70
	case model.DesiredStateProgressIrrelevant:
		return 80
	case model.DesiredStateProgressUnknown:
		return 90
	}
	// should not happen, only as a fallback or if forgotten in the future.
	return 1000
}

func aggregateProcessesHealthForWorkload(ctx context.Context, workloadId *model.K8sWorkloadID, optionalPodManifestInjectionContainerNames map[string]struct{}) (*model.DesiredConditionStatus, error) {
	l := loaders.For(ctx)
	pods, err := l.GetWorkloadPods(ctx, *workloadId)
	if err != nil {
		return nil, err
	}

	foundAgentOnAnyContainer := false
	foundReadyInstrumentedContainer := false
	foundExpectedInstrumentationInstances := false
	containersWithMissingInstances := false
	numUnhealthyProcesses := 0
	numStartingProcesses := 0
	numHealthyProcesses := 0
	for _, pod := range pods {
		for _, container := range pod.Containers {
			_, containerPodManifestInjectionOptional := optionalPodManifestInjectionContainerNames[container.ContainerName]
			if containerPodManifestInjectionOptional {
				foundAgentOnAnyContainer = true
			} else {
				if container.OtelDistroName == nil {
					continue
				}
				foundAgentOnAnyContainer = true
				if !container.ExpectingInstrumentationInstances {
					continue
				}
			}

			foundExpectedInstrumentationInstances = true

			if !container.IsReady {
				// only take into account containers that are ready
				// starting containers might not have the agent ready yet
				continue
			}
			foundReadyInstrumentedContainer = true

			containerId := loaders.PodContainerId{
				Namespace:     pod.PodNamespace,
				PodName:       pod.PodName,
				ContainerName: container.ContainerName,
			}
			podIIs, err := l.GetInstrumentationInstancesForContainer(ctx, containerId)
			if err != nil {
				return nil, err
			}

			if len(podIIs) == 0 {
				// expecting instrumentation instances, but none are found.
				containersWithMissingInstances = true
				continue
			}

			for _, instrumentationInstance := range podIIs {
				if instrumentationInstance.Status.Healthy == nil {
					numStartingProcesses++
				} else if *instrumentationInstance.Status.Healthy {
					numHealthyProcesses++
				} else {
					numUnhealthyProcesses++
				}
			}
		}
	}

	// check for any unhealthy first, regardless of any other conditions
	if numUnhealthyProcesses > 0 {
		reasonStr := string(status.ProcessesHealthStatusReasonSomeUnhealthy)
		return &model.DesiredConditionStatus{
			Name:       status.ProcessesHealthStatusName,
			Status:     model.DesiredStateProgressFailure,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("Found %d processes with unhealthy agent", numUnhealthyProcesses),
		}, nil
	}

	if !foundAgentOnAnyContainer {
		reasonStr := string(status.ProcessesHealthStatusReasonNoAgentInjected)
		return &model.DesiredConditionStatus{
			Name:       status.ProcessesHealthStatusName,
			Status:     model.DesiredStateProgressIrrelevant,
			ReasonEnum: &reasonStr,
			Message:    "none of the running pods is instrumented with odigos agent",
		}, nil
	}

	if !foundExpectedInstrumentationInstances {
		reasonStr := string(status.ProcessesHealthStatusReasonUnsupported)
		return &model.DesiredConditionStatus{
			Name:       status.ProcessesHealthStatusName,
			Status:     model.DesiredStateProgressIrrelevant,
			ReasonEnum: &reasonStr,
			Message:    "agents used in this workload does not support health status reporting",
		}, nil
	}

	if !foundReadyInstrumentedContainer {
		reasonStr := string(status.ProcessesHealthStatusReasonContainersNotReady)
		return &model.DesiredConditionStatus{
			Name:       status.ProcessesHealthStatusName,
			Status:     model.DesiredStateProgressWaiting, // for container to start and become ready
			ReasonEnum: &reasonStr,
			Message:    "agent not yet started in any instrumented containers",
		}, nil
	}

	if containersWithMissingInstances {
		reasonStr := string(status.ProcessesHealthStatusReasonNoProcesses)
		return &model.DesiredConditionStatus{
			Name:       status.ProcessesHealthStatusName,
			Status:     model.DesiredStateProgressWaiting, // instance expected to show up soon
			ReasonEnum: &reasonStr,
			Message:    "agent not yet started in all instrumented containers",
		}, nil
	}

	if numStartingProcesses > 0 {
		reasonStr := string(status.ProcessesHealthStatusReasonStarting)
		return &model.DesiredConditionStatus{
			Name:       status.ProcessesHealthStatusName,
			Status:     model.DesiredStateProgressWaiting,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("Found %d processes with starting agent", numStartingProcesses),
		}, nil
	}

	if numHealthyProcesses > 0 {
		reasonStr := string(status.ProcessesHealthStatusReasonAllHealthy)
		return &model.DesiredConditionStatus{
			Name:       status.ProcessesHealthStatusName,
			Status:     model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr,
			Message:    fmt.Sprintf("All %d agents in instrumented processes are healthy", numHealthyProcesses),
		}, nil
	}

	return nil, nil
}

func aggregateConditionsBySeverity(conditions []*model.DesiredConditionStatus) *model.DesiredConditionStatus {
	var mostSevereCondition *model.DesiredConditionStatus
	for _, condition := range conditions {
		if condition == nil {
			continue
		}
		if mostSevereCondition == nil || desiredStateProgressSeverity(condition.Status) < desiredStateProgressSeverity(mostSevereCondition.Status) {
			mostSevereCondition = condition
		}
	}
	return mostSevereCondition
}

func getContainerNamesWithOptionalPodManifestInjection(ic *v1alpha1.InstrumentationConfig) map[string]struct{} {
	containerNamesWithOptionalPodManifestInjection := map[string]struct{}{}
	if ic == nil {
		return containerNamesWithOptionalPodManifestInjection
	}

	if ic.Spec.PodManifestInjectionOptional {
		for _, container := range ic.Spec.Containers {
			if container.PodManifestInjectionOptional {
				containerNamesWithOptionalPodManifestInjection[container.ContainerName] = struct{}{}
			}
		}
	}
	return containerNamesWithOptionalPodManifestInjection
}
