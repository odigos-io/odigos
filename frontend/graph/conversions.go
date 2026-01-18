package graph

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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
	case "rollout":
		return model.K8sResourceKindRollout
	case "staticpod":
		return model.K8sResourceKindStaticPod
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

func instrumentationConfigToActualSource(ctx context.Context, instruConfig v1alpha1.InstrumentationConfig, dataStreamNames []*string, manifestYAML string) (*model.K8sActualSource, error) {
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

	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(instruConfig.Name, instruConfig.Namespace)
	if err != nil {
		return nil, err
	}

	return &model.K8sActualSource{
		Namespace:         pw.Namespace,
		Kind:              kindToGql(string(pw.Kind)),
		Name:              pw.Name,
		Selected:          &selected,
		DataStreamNames:   dataStreamNames,
		OtelServiceName:   &instruConfig.Spec.ServiceName,
		NumberOfInstances: nil,
		Containers:        containers,
		Conditions:        services.ConvertConditions(instruConfig.Status.Conditions),
		ManifestYaml:      &manifestYAML,
	}, nil
}

func RemoteConfigToModel(config *common.OdigosConfiguration) *model.RemoteConfig {
	if config == nil {
		return nil
	}

	result := &model.RemoteConfig{}
	if config.Rollout != nil {
		result.Rollout = &model.RemoteConfigRollout{
			AutomaticRolloutDisabled: config.Rollout.AutomaticRolloutDisabled,
		}
	}
	return result
}

func EffectiveConfigToModel(config *common.OdigosConfiguration) *model.EffectiveConfig {
	if config == nil {
		return nil
	}

	result := &model.EffectiveConfig{
		ConfigVersion:     config.ConfigVersion,
		IgnoredNamespaces: config.IgnoredNamespaces,
		IgnoredContainers: config.IgnoredContainers,
	}

	// Boolean values
	telemetryEnabled := config.TelemetryEnabled
	result.TelemetryEnabled = &telemetryEnabled

	openshiftEnabled := config.OpenshiftEnabled
	result.OpenshiftEnabled = &openshiftEnabled

	psp := config.Psp
	result.Psp = &psp

	skipWebhookIssuerCreation := config.SkipWebhookIssuerCreation
	result.SkipWebhookIssuerCreation = &skipWebhookIssuerCreation

	// Pointer booleans
	result.IgnoreOdigosNamespace = config.IgnoreOdigosNamespace
	result.AllowConcurrentAgents = config.AllowConcurrentAgents
	result.KarpenterEnabled = config.KarpenterEnabled
	result.RollbackDisabled = config.RollbackDisabled
	result.ClickhouseJSONTypeEnabled = config.ClickhouseJsonTypeEnabledProperty
	result.CheckDeviceHealthBeforeInjection = config.CheckDeviceHealthBeforeInjection
	result.WaspEnabled = config.WaspEnabled

	// String values (only set if non-empty)
	if config.ImagePrefix != "" {
		result.ImagePrefix = &config.ImagePrefix
	}
	if config.UiRemoteUrl != "" {
		result.UIRemoteURL = &config.UiRemoteUrl
	}
	if config.CentralBackendURL != "" {
		result.CentralBackendURL = &config.CentralBackendURL
	}
	if config.ClusterName != "" {
		result.ClusterName = &config.ClusterName
	}
	if config.CustomContainerRuntimeSocketPath != "" {
		result.CustomContainerRuntimeSocketPath = &config.CustomContainerRuntimeSocketPath
	}
	if config.RollbackGraceTime != "" {
		result.RollbackGraceTime = &config.RollbackGraceTime
	}
	if config.RollbackStabilityWindow != "" {
		result.RollbackStabilityWindow = &config.RollbackStabilityWindow
	}
	if config.GoAutoOffsetsCron != "" {
		result.GoAutoOffsetsCron = &config.GoAutoOffsetsCron
	}
	if config.GoAutoOffsetsMode != "" {
		result.GoAutoOffsetsMode = &config.GoAutoOffsetsMode
	}
	if config.ResourceSizePreset != "" {
		result.ResourceSizePreset = &config.ResourceSizePreset
	}
	if config.TraceIdSuffix != "" {
		result.TraceIDSuffix = &config.TraceIdSuffix
	}

	// Int values (only set if non-zero)
	if config.UiPaginationLimit != 0 {
		result.UIPaginationLimit = &config.UiPaginationLimit
	}
	if config.OdigletHealthProbeBindPort != 0 {
		result.OdigletHealthProbeBindPort = &config.OdigletHealthProbeBindPort
	}

	// String arrays
	result.AllowedTestConnectionHosts = config.AllowedTestConnectionHosts
	result.ImagePullSecrets = config.ImagePullSecrets

	// Profiles (convert ProfileName to string)
	if len(config.Profiles) > 0 {
		profiles := make([]string, len(config.Profiles))
		for i, p := range config.Profiles {
			profiles[i] = string(p)
		}
		result.Profiles = profiles
	}

	// UiMode
	if config.UiMode != "" {
		uiMode := convertUiModeToModel(config.UiMode)
		result.UIMode = &uiMode
	}

	// MountMethod
	if config.MountMethod != nil {
		mountMethod := convertMountMethodToModel(*config.MountMethod)
		result.MountMethod = &mountMethod
	}

	// AgentEnvVarsInjectionMethod
	if config.AgentEnvVarsInjectionMethod != nil {
		injectionMethod := convertEnvInjectionMethodToModel(*config.AgentEnvVarsInjectionMethod)
		result.AgentEnvVarsInjectionMethod = &injectionMethod
	}

	// NodeSelector (convert map to JSON string)
	if len(config.NodeSelector) > 0 {
		nodeSelectorJSON, _ := json.Marshal(config.NodeSelector)
		nodeSelectorStr := string(nodeSelectorJSON)
		result.NodeSelector = &nodeSelectorStr
	}

	// CollectorGateway
	if config.CollectorGateway != nil {
		result.CollectorGateway = convertCollectorGatewayToModel(config.CollectorGateway)
	}

	// CollectorNode
	if config.CollectorNode != nil {
		result.CollectorNode = convertCollectorNodeToModel(config.CollectorNode)
	}

	// Rollout
	if config.Rollout != nil {
		result.Rollout = &model.RolloutConfig{
			AutomaticRolloutDisabled: config.Rollout.AutomaticRolloutDisabled,
		}
	}

	// Oidc
	if config.Oidc != nil {
		result.Oidc = &model.OidcConfig{}
		if config.Oidc.TenantUrl != "" {
			result.Oidc.TenantURL = &config.Oidc.TenantUrl
		}
		if config.Oidc.ClientId != "" {
			result.Oidc.ClientID = &config.Oidc.ClientId
		}
		if config.Oidc.ClientSecret != "" {
			result.Oidc.ClientSecret = &config.Oidc.ClientSecret
		}
	}

	// UserInstrumentationEnvs
	if config.UserInstrumentationEnvs != nil {
		result.UserInstrumentationEnvs = convertUserInstrumentationEnvsToModel(config.UserInstrumentationEnvs)
	}

	// MetricsSources
	if config.MetricsSources != nil {
		result.MetricsSources = convertMetricsSourcesToModel(config.MetricsSources)
	}

	// AgentsInitContainerResources
	if config.AgentsInitContainerResources != nil {
		result.AgentsInitContainerResources = &model.AgentsInitContainerResourcesConfig{}
		if config.AgentsInitContainerResources.RequestCPUm != 0 {
			result.AgentsInitContainerResources.RequestCPUm = &config.AgentsInitContainerResources.RequestCPUm
		}
		if config.AgentsInitContainerResources.LimitCPUm != 0 {
			result.AgentsInitContainerResources.LimitCPUm = &config.AgentsInitContainerResources.LimitCPUm
		}
		if config.AgentsInitContainerResources.RequestMemoryMiB != 0 {
			result.AgentsInitContainerResources.RequestMemoryMiB = &config.AgentsInitContainerResources.RequestMemoryMiB
		}
		if config.AgentsInitContainerResources.LimitMemoryMiB != 0 {
			result.AgentsInitContainerResources.LimitMemoryMiB = &config.AgentsInitContainerResources.LimitMemoryMiB
		}
	}

	// OdigosOwnTelemetryStore
	if config.OdigosOwnTelemetryStore != nil {
		result.OdigosOwnTelemetryStore = &model.OdigosOwnTelemetryConfig{
			MetricsStoreDisabled: config.OdigosOwnTelemetryStore.MetricsStoreDisabled,
		}
	}

	return result
}

func convertUiModeToModel(uiMode common.UiMode) model.UIMode {
	switch uiMode {
	case common.UiModeReadonly:
		return model.UIModeReadonly
	default:
		return model.UIModeDefault
	}
}

func convertMountMethodToModel(method common.MountMethod) model.MountMethod {
	switch method {
	case common.K8sHostPathMountMethod:
		return model.MountMethodK8sHostPath
	case common.K8sInitContainerMountMethod:
		return model.MountMethodK8sInitContainer
	default:
		return model.MountMethodK8sVirtualDevice
	}
}

func convertEnvInjectionMethodToModel(method common.EnvInjectionMethod) model.EnvInjectionMethod {
	switch method {
	case common.PodManifestEnvInjectionMethod:
		return model.EnvInjectionMethodPodManifest
	case common.LoaderFallbackToPodManifestInjectionMethod:
		return model.EnvInjectionMethodLoaderFallbackToPodManifest
	default:
		return model.EnvInjectionMethodLoader
	}
}

func convertCollectorGatewayToModel(gw *common.CollectorGatewayConfiguration) *model.CollectorGatewayConfig {
	if gw == nil {
		return nil
	}

	result := &model.CollectorGatewayConfig{}

	if gw.MinReplicas != 0 {
		result.MinReplicas = &gw.MinReplicas
	}
	if gw.MaxReplicas != 0 {
		result.MaxReplicas = &gw.MaxReplicas
	}
	if gw.RequestMemoryMiB != 0 {
		result.RequestMemoryMiB = &gw.RequestMemoryMiB
	}
	if gw.LimitMemoryMiB != 0 {
		result.LimitMemoryMiB = &gw.LimitMemoryMiB
	}
	if gw.RequestCPUm != 0 {
		result.RequestCPUm = &gw.RequestCPUm
	}
	if gw.LimitCPUm != 0 {
		result.LimitCPUm = &gw.LimitCPUm
	}
	if gw.MemoryLimiterLimitMiB != 0 {
		result.MemoryLimiterLimitMiB = &gw.MemoryLimiterLimitMiB
	}
	if gw.MemoryLimiterSpikeLimitMiB != 0 {
		result.MemoryLimiterSpikeLimitMiB = &gw.MemoryLimiterSpikeLimitMiB
	}
	if gw.GoMemLimitMib != 0 {
		result.GoMemLimitMiB = &gw.GoMemLimitMib
	}
	result.ServiceGraphDisabled = gw.ServiceGraphDisabled
	result.ClusterMetricsEnabled = gw.ClusterMetricsEnabled
	result.HTTPSProxyAddress = gw.HttpsProxyAddress

	if gw.NodeSelector != nil && len(*gw.NodeSelector) > 0 {
		nodeSelectorJSON, _ := json.Marshal(*gw.NodeSelector)
		nodeSelectorStr := string(nodeSelectorJSON)
		result.NodeSelector = &nodeSelectorStr
	}

	return result
}

func convertCollectorNodeToModel(node *common.CollectorNodeConfiguration) *model.CollectorNodeConfig {
	if node == nil {
		return nil
	}

	result := &model.CollectorNodeConfig{}

	if node.CollectorOwnMetricsPort != 0 {
		port := int(node.CollectorOwnMetricsPort)
		result.CollectorOwnMetricsPort = &port
	}
	if node.RequestMemoryMiB != 0 {
		result.RequestMemoryMiB = &node.RequestMemoryMiB
	}
	if node.LimitMemoryMiB != 0 {
		result.LimitMemoryMiB = &node.LimitMemoryMiB
	}
	if node.RequestCPUm != 0 {
		result.RequestCPUm = &node.RequestCPUm
	}
	if node.LimitCPUm != 0 {
		result.LimitCPUm = &node.LimitCPUm
	}
	if node.MemoryLimiterLimitMiB != 0 {
		result.MemoryLimiterLimitMiB = &node.MemoryLimiterLimitMiB
	}
	if node.MemoryLimiterSpikeLimitMiB != 0 {
		result.MemoryLimiterSpikeLimitMiB = &node.MemoryLimiterSpikeLimitMiB
	}
	if node.GoMemLimitMib != 0 {
		result.GoMemLimitMiB = &node.GoMemLimitMib
	}
	if node.K8sNodeLogsDirectory != "" {
		result.K8sNodeLogsDirectory = &node.K8sNodeLogsDirectory
	}
	result.EnableDataCompression = node.EnableDataCompression

	if node.OtlpExporterConfiguration != nil {
		result.OtlpExporterConfiguration = convertOtlpExporterToModel(node.OtlpExporterConfiguration)
	}

	return result
}

func convertOtlpExporterToModel(otlp *common.OtlpExporterConfiguration) *model.OtlpExporterConfig {
	if otlp == nil {
		return nil
	}

	result := &model.OtlpExporterConfig{
		EnableDataCompression: otlp.EnableDataCompression,
	}

	if otlp.Timeout != "" {
		result.Timeout = &otlp.Timeout
	}

	if otlp.RetryOnFailure != nil {
		result.RetryOnFailure = &model.RetryOnFailureConfig{
			Enabled: otlp.RetryOnFailure.Enabled,
		}
		if otlp.RetryOnFailure.InitialInterval != "" {
			result.RetryOnFailure.InitialInterval = &otlp.RetryOnFailure.InitialInterval
		}
		if otlp.RetryOnFailure.MaxInterval != "" {
			result.RetryOnFailure.MaxInterval = &otlp.RetryOnFailure.MaxInterval
		}
		if otlp.RetryOnFailure.MaxElapsedTime != "" {
			result.RetryOnFailure.MaxElapsedTime = &otlp.RetryOnFailure.MaxElapsedTime
		}
	}

	return result
}

func convertUserInstrumentationEnvsToModel(envs *common.UserInstrumentationEnvs) *model.UserInstrumentationEnvsConfig {
	if envs == nil {
		return nil
	}

	result := &model.UserInstrumentationEnvsConfig{}

	if len(envs.Languages) > 0 {
		languagesJSON, _ := json.Marshal(envs.Languages)
		languagesStr := string(languagesJSON)
		result.Languages = &languagesStr
	}

	return result
}

func convertMetricsSourcesToModel(ms *common.MetricsSourceConfiguration) *model.MetricsSourceConfig {
	if ms == nil {
		return nil
	}

	result := &model.MetricsSourceConfig{}

	if ms.SpanMetrics != nil {
		sm := ms.SpanMetrics
		result.SpanMetrics = &model.MetricsSourceSpanMetricsConfig{
			Disabled:                     sm.Disabled,
			AdditionalDimensions:         sm.AdditionalDimensions,
			HistogramBuckets:             sm.ExplicitHistogramBuckets,
			IncludedProcessInDimensions:  sm.IncludedProcessInDimensions,
			ExcludedResourceAttributes:   sm.ExcludedResourceAttributes,
			ResourceMetricsKeyAttributes: sm.ResourceMetricsKeyAttributes,
		}
		if sm.Interval != "" {
			result.SpanMetrics.Interval = &sm.Interval
		}
		if sm.MetricsExpiration != "" {
			result.SpanMetrics.MetricsExpiration = &sm.MetricsExpiration
		}
		if sm.HistogramDisabled {
			result.SpanMetrics.HistogramDisabled = &sm.HistogramDisabled
		}
	}

	if ms.HostMetrics != nil {
		result.HostMetrics = &model.MetricsSourceHostMetricsConfig{
			Disabled: ms.HostMetrics.Disabled,
		}
		if ms.HostMetrics.Interval != "" {
			result.HostMetrics.Interval = &ms.HostMetrics.Interval
		}
	}

	if ms.KubeletStats != nil {
		result.KubeletStats = &model.MetricsSourceKubeletStatsConfig{
			Disabled: ms.KubeletStats.Disabled,
		}
		if ms.KubeletStats.Interval != "" {
			result.KubeletStats.Interval = &ms.KubeletStats.Interval
		}
	}

	if ms.OdigosOwnMetrics != nil {
		result.OdigosOwnMetrics = &model.MetricsSourceOdigosOwnMetricsConfig{}
		if ms.OdigosOwnMetrics.Interval != "" {
			result.OdigosOwnMetrics.Interval = &ms.OdigosOwnMetrics.Interval
		}
	}

	if ms.AgentMetrics != nil {
		result.AgentMetrics = &model.MetricsSourceAgentMetricsConfig{}

		if ms.AgentMetrics.SpanMetrics != nil {
			result.AgentMetrics.SpanMetrics = &model.MetricsSourceAgentSpanMetricsConfig{
				Enabled: ms.AgentMetrics.SpanMetrics.Enabled,
			}
		}

		if ms.AgentMetrics.RuntimeMetrics != nil && ms.AgentMetrics.RuntimeMetrics.Java != nil {
			result.AgentMetrics.RuntimeMetrics = &model.MetricsSourceAgentRuntimeMetricsConfig{
				Java: &model.MetricsSourceAgentJavaRuntimeMetricsConfig{
					Disabled: ms.AgentMetrics.RuntimeMetrics.Java.Disabled,
				},
			}

			if len(ms.AgentMetrics.RuntimeMetrics.Java.Metrics) > 0 {
				metrics := make([]*model.MetricsSourceAgentRuntimeMetricConfig, len(ms.AgentMetrics.RuntimeMetrics.Java.Metrics))
				for i, m := range ms.AgentMetrics.RuntimeMetrics.Java.Metrics {
					metrics[i] = &model.MetricsSourceAgentRuntimeMetricConfig{
						Name:     m.Name,
						Disabled: m.Disabled,
					}
				}
				result.AgentMetrics.RuntimeMetrics.Java.Metrics = metrics
			}
		}
	}

	return result
}
