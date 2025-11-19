package graph

import (
	"context"
	"strings"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"
)

func kindToGql(kind string) model.K8sResourceKind {
	switch strings.ToLower(kind) {
	case "deployment":
		return model.K8sResourceKindDeployment
	case "statefulset":
		return model.K8sResourceKindStatefulSet
	case "daemonset":
		return model.K8sResourceKindDaemonSet
	case "cronjob":
		return model.K8sResourceKindCronJob
	case "deploymentconfig":
		return model.K8sResourceKindDeploymentConfig
	}
	return ""
}

func getContainerAgentInfo(ic *v1alpha1.InstrumentationConfig, containerName string) (bool, string, string) {
	for _, specContainer := range ic.Spec.Containers {
		if specContainer.ContainerName == containerName {
			instrumented := specContainer.AgentEnabled
			instrumentationMessage := specContainer.AgentEnabledMessage
			if instrumentationMessage == "" {
				instrumentationMessage = string(specContainer.AgentEnabledReason)
			}
			otelDistroName := specContainer.OtelDistroName
			return instrumented, instrumentationMessage, otelDistroName
		}
	}
	return false, "", ""
}

func instrumentationConfigToActualSource(ctx context.Context, instruConfig v1alpha1.InstrumentationConfig, dataStreamNames []*string) (*model.K8sActualSource, error) {
	selected := true
	var containers []*model.SourceContainer

	// Map the containers runtime details
	for i := range instruConfig.Status.RuntimeDetailsByContainer {
		statusContainer := instruConfig.Status.RuntimeDetailsByContainer[i]
		containerName := statusContainer.ContainerName
		instrumented, instrumentationMessage, otelDistroName := getContainerAgentInfo(&instruConfig, containerName)

		resolvedRuntimeInfo := &statusContainer
		overriden := false
		for _, override := range instruConfig.Spec.ContainersOverrides {
			if override.ContainerName == containerName {
				if override.RuntimeInfo != nil {
					resolvedRuntimeInfo = override.RuntimeInfo
					overriden = true
				}
				break
			}
		}

		containers = append(containers, &model.SourceContainer{
			ContainerName:          containerName,
			Language:               string(resolvedRuntimeInfo.Language),
			RuntimeVersion:         resolvedRuntimeInfo.RuntimeVersion,
			Overriden:              overriden,
			Instrumented:           instrumented,
			InstrumentationMessage: instrumentationMessage,
			OtelDistroName:         &otelDistroName,
		})
	}

	if len(containers) == 0 {
		// then take the containers from the overrides
		for _, override := range instruConfig.Spec.ContainersOverrides {
			language := ""
			if override.RuntimeInfo != nil {
				language = string(override.RuntimeInfo.Language)
			}
			runtimeVersion := ""
			if override.RuntimeInfo != nil {
				runtimeVersion = override.RuntimeInfo.RuntimeVersion
			}
			instrumented, instrumentationMessage, otelDistroName := getContainerAgentInfo(&instruConfig, override.ContainerName)

			containers = append(containers, &model.SourceContainer{
				ContainerName:          override.ContainerName,
				Language:               language,
				RuntimeVersion:         runtimeVersion,
				Overriden:              true,
				Instrumented:           instrumented,
				InstrumentationMessage: instrumentationMessage,
				OtelDistroName:         &otelDistroName,
			})
		}
	}

	// Return the converted K8sActualSource object
	return &model.K8sActualSource{
		Namespace:         instruConfig.Namespace,
		Kind:              kindToGql(instruConfig.OwnerReferences[0].Kind),
		Name:              instruConfig.OwnerReferences[0].Name,
		Selected:          &selected,
		DataStreamNames:   dataStreamNames,
		OtelServiceName:   &instruConfig.Spec.ServiceName,
		NumberOfInstances: nil,
		Containers:        containers,
		Conditions:        services.ConvertConditions(instruConfig.Status.Conditions),
	}, nil
}
