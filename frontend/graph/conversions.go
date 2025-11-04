package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
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

func convertOdigosConfigToK8s(cfg *model.OdigosConfigurationInput) (*common.OdigosConfiguration, error) {
	odigosConfig := common.OdigosConfiguration{}

	if cfg.KarpenterEnabled != nil {
		odigosConfig.KarpenterEnabled = cfg.KarpenterEnabled
	}
	if cfg.AllowConcurrentAgents != nil {
		odigosConfig.AllowConcurrentAgents = cfg.AllowConcurrentAgents
	}
	if cfg.UIPaginationLimit != nil {
		odigosConfig.UiPaginationLimit = *cfg.UIPaginationLimit
	}
	if cfg.CentralBackendURL != nil {
		odigosConfig.CentralBackendURL = *cfg.CentralBackendURL
	}
	if cfg.Oidc != nil {
		odigosConfig.Oidc = &common.OidcConfiguration{}
		if cfg.Oidc.TenantURL != nil {
			odigosConfig.Oidc.TenantUrl = *cfg.Oidc.TenantURL
		}
		if cfg.Oidc.ClientID != nil {
			odigosConfig.Oidc.ClientId = *cfg.Oidc.ClientID
		}
		if cfg.Oidc.ClientSecret != nil {
			odigosConfig.Oidc.ClientSecret = *cfg.Oidc.ClientSecret
		}
	}
	if cfg.ClusterName != nil {
		odigosConfig.ClusterName = *cfg.ClusterName
	}
	if cfg.ImagePrefix != nil {
		odigosConfig.ImagePrefix = *cfg.ImagePrefix
	}
	if cfg.IgnoredNamespaces != nil {
		odigosConfig.IgnoredNamespaces = make([]string, len(cfg.IgnoredNamespaces))
		for i, ns := range cfg.IgnoredNamespaces {
			odigosConfig.IgnoredNamespaces[i] = *ns
		}
	}
	if cfg.IgnoredContainers != nil {
		odigosConfig.IgnoredContainers = make([]string, len(cfg.IgnoredContainers))
		for i, cont := range cfg.IgnoredContainers {
			odigosConfig.IgnoredContainers[i] = *cont
		}
	}
	if len(cfg.Profiles) > 0 {
		odigosConfig.Profiles = make([]common.ProfileName, len(cfg.Profiles))
		for i := range cfg.Profiles {
			odigosConfig.Profiles[i] = common.ProfileName(*cfg.Profiles[i])
		}
	}
	if cfg.MountMethod != nil {
		m := common.MountMethod(*cfg.MountMethod)
		odigosConfig.MountMethod = &m
	}
	if cfg.AgentEnvVarsInjectionMethod != nil {
		m := common.EnvInjectionMethod(*cfg.AgentEnvVarsInjectionMethod)
		odigosConfig.AgentEnvVarsInjectionMethod = &m
	}
	if cfg.CustomContainerRuntimeSocketPath != nil {
		odigosConfig.CustomContainerRuntimeSocketPath = *cfg.CustomContainerRuntimeSocketPath
	}
	if cfg.OdigletHealthProbeBindPort != nil {
		odigosConfig.OdigletHealthProbeBindPort = *cfg.OdigletHealthProbeBindPort
	}
	if cfg.RollbackDisabled != nil {
		odigosConfig.RollbackDisabled = cfg.RollbackDisabled
	}
	if cfg.RollbackGraceTime != nil {
		odigosConfig.RollbackGraceTime = *cfg.RollbackGraceTime
	}
	if cfg.RollbackStabilityWindow != nil {
		odigosConfig.RollbackStabilityWindow = *cfg.RollbackStabilityWindow
	}
	if cfg.Rollout != nil {
		odigosConfig.Rollout = &common.RolloutConfiguration{
			AutomaticRolloutDisabled: cfg.Rollout.AutomaticRolloutDisabled,
		}
	}
	if cfg.NodeSelector != nil {
		var jsonParsed map[string]string
		err := json.Unmarshal([]byte(*cfg.NodeSelector), &jsonParsed)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal NodeSelector: %v", err)
		}
		odigosConfig.NodeSelector = jsonParsed
	}
	if cfg.CollectorNode != nil {
		odigosConfig.CollectorNode = &common.CollectorNodeConfiguration{}
		if cfg.CollectorNode.CollectorOwnMetricsPort != nil {
			odigosConfig.CollectorNode.CollectorOwnMetricsPort = int32(*cfg.CollectorNode.CollectorOwnMetricsPort)
		}
		if cfg.CollectorNode.RequestMemoryMiB != nil {
			odigosConfig.CollectorNode.RequestMemoryMiB = *cfg.CollectorNode.RequestMemoryMiB
		}
		if cfg.CollectorNode.LimitMemoryMiB != nil {
			odigosConfig.CollectorNode.LimitMemoryMiB = *cfg.CollectorNode.LimitMemoryMiB
		}
		if cfg.CollectorNode.RequestCPUm != nil {
			odigosConfig.CollectorNode.RequestCPUm = *cfg.CollectorNode.RequestCPUm
		}
		if cfg.CollectorNode.LimitCPUm != nil {
			odigosConfig.CollectorNode.LimitCPUm = *cfg.CollectorNode.LimitCPUm
		}
		if cfg.CollectorNode.MemoryLimiterLimitMiB != nil {
			odigosConfig.CollectorNode.MemoryLimiterLimitMiB = *cfg.CollectorNode.MemoryLimiterLimitMiB
		}
		if cfg.CollectorNode.MemoryLimiterSpikeLimitMiB != nil {
			odigosConfig.CollectorNode.MemoryLimiterSpikeLimitMiB = *cfg.CollectorNode.MemoryLimiterSpikeLimitMiB
		}
		if cfg.CollectorNode.GoMemLimitMiB != nil {
			odigosConfig.CollectorNode.GoMemLimitMib = *cfg.CollectorNode.GoMemLimitMiB
		}
		if cfg.CollectorNode.K8sNodeLogsDirectory != nil {
			odigosConfig.CollectorNode.K8sNodeLogsDirectory = *cfg.CollectorNode.K8sNodeLogsDirectory
		}
	}
	if cfg.CollectorGateway != nil {
		odigosConfig.CollectorGateway = &common.CollectorGatewayConfiguration{}
		if cfg.CollectorGateway.MinReplicas != nil {
			odigosConfig.CollectorGateway.MinReplicas = *cfg.CollectorGateway.MinReplicas
		}
		if cfg.CollectorGateway.MaxReplicas != nil {
			odigosConfig.CollectorGateway.MaxReplicas = *cfg.CollectorGateway.MaxReplicas
		}
		if cfg.CollectorGateway.RequestMemoryMiB != nil {
			odigosConfig.CollectorGateway.RequestMemoryMiB = *cfg.CollectorGateway.RequestMemoryMiB
		}
		if cfg.CollectorGateway.LimitMemoryMiB != nil {
			odigosConfig.CollectorGateway.LimitMemoryMiB = *cfg.CollectorGateway.LimitMemoryMiB
		}
		if cfg.CollectorGateway.RequestCPUm != nil {
			odigosConfig.CollectorGateway.RequestCPUm = *cfg.CollectorGateway.RequestCPUm
		}
		if cfg.CollectorGateway.LimitCPUm != nil {
			odigosConfig.CollectorGateway.LimitCPUm = *cfg.CollectorGateway.LimitCPUm
		}
		if cfg.CollectorGateway.MemoryLimiterLimitMiB != nil {
			odigosConfig.CollectorGateway.MemoryLimiterLimitMiB = *cfg.CollectorGateway.MemoryLimiterLimitMiB
		}
		if cfg.CollectorGateway.MemoryLimiterSpikeLimitMiB != nil {
			odigosConfig.CollectorGateway.MemoryLimiterSpikeLimitMiB = *cfg.CollectorGateway.MemoryLimiterSpikeLimitMiB
		}
		if cfg.CollectorGateway.GoMemLimitMiB != nil {
			odigosConfig.CollectorGateway.GoMemLimitMib = *cfg.CollectorGateway.GoMemLimitMiB
		}
	}

	return &odigosConfig, nil
}

func convertOdigosConfigToGql(cfg *common.OdigosConfiguration) (*model.OdigosConfiguration, error) {
	odigosConfig := model.OdigosConfiguration{
		KarpenterEnabled:                 cfg.KarpenterEnabled,
		AllowConcurrentAgents:            cfg.AllowConcurrentAgents,
		ClusterName:                      &cfg.ClusterName,
		ImagePrefix:                      &cfg.ImagePrefix,
		CentralBackendURL:                &cfg.CentralBackendURL,
		RollbackDisabled:                 cfg.RollbackDisabled,
		RollbackGraceTime:                &cfg.RollbackGraceTime,
		RollbackStabilityWindow:          &cfg.RollbackStabilityWindow,
		CustomContainerRuntimeSocketPath: &cfg.CustomContainerRuntimeSocketPath,
	}

	if cfg.UiPaginationLimit > 0 {
		odigosConfig.UIPaginationLimit = &cfg.UiPaginationLimit
	}
	if len(cfg.Profiles) > 0 {
		odigosConfig.Profiles = make([]*string, len(cfg.Profiles))
		for i := range cfg.Profiles {
			profile := string(cfg.Profiles[i])
			odigosConfig.Profiles[i] = &profile
		}
	}
	if cfg.MountMethod != nil {
		mountMethod := string(*cfg.MountMethod)
		odigosConfig.MountMethod = &mountMethod
	}
	if cfg.AgentEnvVarsInjectionMethod != nil {
		agentEnvVarsInjectionMethod := string(*cfg.AgentEnvVarsInjectionMethod)
		odigosConfig.AgentEnvVarsInjectionMethod = &agentEnvVarsInjectionMethod
	}
	if cfg.OdigletHealthProbeBindPort > 0 {
		port := int(cfg.OdigletHealthProbeBindPort)
		odigosConfig.OdigletHealthProbeBindPort = &port
	}
	if len(cfg.IgnoredNamespaces) > 0 {
		ignoredNamespaces := make([]*string, len(cfg.IgnoredNamespaces))
		for i, ns := range cfg.IgnoredNamespaces {
			ignoredNamespaces[i] = &ns
		}
		odigosConfig.IgnoredNamespaces = ignoredNamespaces
	}
	if len(cfg.IgnoredContainers) > 0 {
		ignoredContainers := make([]*string, len(cfg.IgnoredContainers))
		for i, container := range cfg.IgnoredContainers {
			ignoredContainers[i] = &container
		}
		odigosConfig.IgnoredContainers = ignoredContainers
	}
	if cfg.Oidc != nil {
		odigosConfig.Oidc = &model.OidcConfiguration{
			TenantURL:    &cfg.Oidc.TenantUrl,
			ClientID:     &cfg.Oidc.ClientId,
			ClientSecret: &cfg.Oidc.ClientSecret,
		}
	}
	if cfg.Rollout != nil {
		odigosConfig.Rollout = &model.RolloutConfiguration{
			AutomaticRolloutDisabled: cfg.Rollout.AutomaticRolloutDisabled,
		}
	}
	if cfg.NodeSelector != nil {
		nodeSelectorBytes, err := json.Marshal(cfg.NodeSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal NodeSelector: %v", err)
		}
		nodeSelectorStr := string(nodeSelectorBytes)
		if nodeSelectorStr != "" {
			odigosConfig.NodeSelector = &nodeSelectorStr
		}
	}

	if cfg.CollectorNode != nil {
		odigosConfig.CollectorNode = &model.CollectorNode{
			K8sNodeLogsDirectory: &cfg.CollectorNode.K8sNodeLogsDirectory,
		}

		if cfg.CollectorNode.CollectorOwnMetricsPort > 0 {
			port := int(cfg.CollectorNode.CollectorOwnMetricsPort)
			odigosConfig.CollectorNode.CollectorOwnMetricsPort = &port
		}
		if cfg.CollectorNode.RequestMemoryMiB > 0 {
			odigosConfig.CollectorNode.RequestMemoryMiB = &cfg.CollectorNode.RequestMemoryMiB
		}
		if cfg.CollectorNode.LimitMemoryMiB > 0 {
			odigosConfig.CollectorNode.LimitMemoryMiB = &cfg.CollectorNode.LimitMemoryMiB
		}
		if cfg.CollectorNode.RequestCPUm > 0 {
			odigosConfig.CollectorNode.RequestCPUm = &cfg.CollectorNode.RequestCPUm
		}
		if cfg.CollectorNode.LimitCPUm > 0 {
			odigosConfig.CollectorNode.LimitCPUm = &cfg.CollectorNode.LimitCPUm
		}
		if cfg.CollectorNode.MemoryLimiterLimitMiB > 0 {
			odigosConfig.CollectorNode.MemoryLimiterLimitMiB = &cfg.CollectorNode.MemoryLimiterLimitMiB
		}
		if cfg.CollectorNode.MemoryLimiterSpikeLimitMiB > 0 {
			odigosConfig.CollectorNode.MemoryLimiterSpikeLimitMiB = &cfg.CollectorNode.MemoryLimiterSpikeLimitMiB
		}
		if cfg.CollectorNode.GoMemLimitMib > 0 {
			odigosConfig.CollectorNode.GoMemLimitMiB = &cfg.CollectorNode.GoMemLimitMib
		}
	}
	if cfg.CollectorGateway != nil {
		odigosConfig.CollectorGateway = &model.CollectorGateway{}

		if cfg.CollectorGateway.RequestMemoryMiB > 0 {
			odigosConfig.CollectorGateway.RequestMemoryMiB = &cfg.CollectorGateway.RequestMemoryMiB
		}
		if cfg.CollectorGateway.LimitMemoryMiB > 0 {
			odigosConfig.CollectorGateway.LimitMemoryMiB = &cfg.CollectorGateway.LimitMemoryMiB
		}
		if cfg.CollectorGateway.RequestCPUm > 0 {
			odigosConfig.CollectorGateway.RequestCPUm = &cfg.CollectorGateway.RequestCPUm
		}
		if cfg.CollectorGateway.LimitCPUm > 0 {
			odigosConfig.CollectorGateway.LimitCPUm = &cfg.CollectorGateway.LimitCPUm
		}
		if cfg.CollectorGateway.MemoryLimiterLimitMiB > 0 {
			odigosConfig.CollectorGateway.MemoryLimiterLimitMiB = &cfg.CollectorGateway.MemoryLimiterLimitMiB
		}
		if cfg.CollectorGateway.MemoryLimiterSpikeLimitMiB > 0 {
			odigosConfig.CollectorGateway.MemoryLimiterSpikeLimitMiB = &cfg.CollectorGateway.MemoryLimiterSpikeLimitMiB
		}
		if cfg.CollectorGateway.GoMemLimitMib > 0 {
			odigosConfig.CollectorGateway.GoMemLimitMiB = &cfg.CollectorGateway.GoMemLimitMib
		}
		if cfg.CollectorGateway.MinReplicas > 0 {
			odigosConfig.CollectorGateway.MinReplicas = &cfg.CollectorGateway.MinReplicas
		}
		if cfg.CollectorGateway.MaxReplicas > 0 {
			odigosConfig.CollectorGateway.MaxReplicas = &cfg.CollectorGateway.MaxReplicas
		}
	}

	return &odigosConfig, nil
}
