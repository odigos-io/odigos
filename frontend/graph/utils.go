package graph

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	corev1 "k8s.io/api/core/v1"
)

const (
	podContainerHealthStatus = "PodCotainerHealth"
	podHealthStatus          = "PodHealth"
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
		AgentEnabled:       true,
		AgentEnabledStatus: status.CalculateAgentInjectionEnabledStatusForContainer(containerAgentConfig),
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
