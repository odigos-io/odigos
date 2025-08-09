package graph

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	corev1 "k8s.io/api/core/v1"
)

const (
	agentInjectedStatus      = "AgentInjected"
	podContainerHealthStatus = "PodCotainerHealth"
	podHealthStatus          = "PodHealth"
	podsAgentInjectionStatus = "PodsAgentInjection"
)

type PodContainerHealthReason string

const (
	PodContainerHealthReasonCrashLoopBackOff PodContainerHealthReason = "CrashLoopBackOff"
	PodContainerHealthReasonNotStarted       PodContainerHealthReason = "NotStarted"
	PodContainerHealthReasonNotReady         PodContainerHealthReason = "NotReady"
	PodContainerHealthReasonHealthy          PodContainerHealthReason = "Healthy"
)

type PodsAgentInjectionReason string

const (
	PodsAgentInjectionReasonNoPodsAgentInjected      PodsAgentInjectionReason = "NoPods"
	PodsAgentInjectionReasonAllPodsAgentInjected     PodsAgentInjectionReason = "AllPodsAgentInjected"
	PodsAgentInjectionReasonAllPodsAgentNotInjected  PodsAgentInjectionReason = "AllPodsAgentNotInjected"
	PodsAgentInjectionReasonSomePodsAgentNotInjected PodsAgentInjectionReason = "SomePodsAgentNotInjected"
	PodsAgentInjectionReasonSomePodsAgentInjected    PodsAgentInjectionReason = "SomePodsAgentInjected"
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
			Key:   name,
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

func runtimeDetectionStatusCondition(reason *string) model.DesiredStateProgress {
	if reason == nil {
		return model.DesiredStateProgressUnknown
	}
	switch v1alpha1.RuntimeDetectionReason(*reason) {
	case v1alpha1.RuntimeDetectionReasonDetectedSuccessfully:
		return model.DesiredStateProgressSuccess
	case v1alpha1.RuntimeDetectionReasonWaitingForDetection:
		return model.DesiredStateProgressWaiting
	case v1alpha1.RuntimeDetectionReasonNoRunningPods:
		return model.DesiredStateProgressPending
	case v1alpha1.RuntimeDetectionReasonError:
		return model.DesiredStateProgressError
	}
	return model.DesiredStateProgressUnknown
}

func agentEnabledStatusCondition(reason *string) model.DesiredStateProgress {
	if reason == nil {
		return model.DesiredStateProgressUnknown
	}
	switch v1alpha1.AgentEnabledReason(*reason) {
	case v1alpha1.AgentEnabledReasonEnabledSuccessfully:
		return model.DesiredStateProgressSuccess
	case v1alpha1.AgentEnabledReasonWaitingForRuntimeInspection:
		return model.DesiredStateProgressWaiting
	case v1alpha1.AgentEnabledReasonWaitingForNodeCollector:
		return model.DesiredStateProgressWaiting
	case v1alpha1.AgentEnabledReasonIgnoredContainer:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonNoCollectedSignals:
		return model.DesiredStateProgressNotice
	case v1alpha1.AgentEnabledReasonUnsupportedProgrammingLanguage:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonNoAvailableAgent:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonInjectionConflict:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonUnsupportedRuntimeVersion:
		return model.DesiredStateProgressDisabled
	case v1alpha1.AgentEnabledReasonMissingDistroParameter:
		return model.DesiredStateProgressError
	case v1alpha1.AgentEnabledReasonOtherAgentDetected:
		return model.DesiredStateProgressNotice
	case v1alpha1.AgentEnabledReasonRuntimeDetailsUnavailable:
		return model.DesiredStateProgressPending
	case v1alpha1.AgentEnabledReasonCrashLoopBackOff:
		return model.DesiredStateProgressError
	}
	return model.DesiredStateProgressUnknown
}

func workloadRolloutStatusCondition(reason *string) model.DesiredStateProgress {
	if reason == nil {
		return model.DesiredStateProgressUnknown
	}
	switch v1alpha1.WorkloadRolloutReason(*reason) {

	case v1alpha1.WorkloadRolloutReasonTriggeredSuccessfully:
		return model.DesiredStateProgressSuccess
	case v1alpha1.WorkloadRolloutReasonFailedToPatch:
		return model.DesiredStateProgressError
	case v1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing:
		return model.DesiredStateProgressWaiting
	case v1alpha1.WorkloadRolloutReasonDisabled:
		return model.DesiredStateProgressDisabled
	case v1alpha1.WorkloadRolloutReasonWaitingForRestart:
		return model.DesiredStateProgressPending
	}
	return model.DesiredStateProgressUnknown
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
	reasonStr := string(containerAgentConfig.AgentEnabledReason)
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
		ContainerName: containerAgentConfig.ContainerName,
		AgentEnabled:  true,
		AgentEnabledStatus: &model.DesiredConditionStatus{
			Name:       v1alpha1.AgentEnabledStatusConditionType,
			Status:     agentEnabledStatusCondition(&reasonStr),
			ReasonEnum: &reasonStr,
		},
		OtelDistroName:     emptyStrToNil(containerAgentConfig.OtelDistroName),
		EnvInjectionMethod: envInjectionMethodStr,
		DistroParams:       distroParamsToModel(containerAgentConfig.DistroParams),
		Traces:             traces,
		Metrics:            metrics,
		Logs:               logs,
	}
}

func getContainerStatus(pod *corev1.Pod, containerName string) *corev1.ContainerStatus {
	for i := range pod.Status.ContainerStatuses {
		containerStatus := &pod.Status.ContainerStatuses[i]
		if containerStatus.Name == containerName {
			return containerStatus
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		containerStatus := &pod.Status.InitContainerStatuses[i]
		if containerStatus.Name == containerName {
			return containerStatus
		}
	}
	return nil
}
