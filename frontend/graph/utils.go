package graph

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
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
