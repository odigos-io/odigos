package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
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

func instrumentationConfigToActualSource(ctx context.Context, instruConfig v1alpha1.InstrumentationConfig, dataStreamNames []*string, manifestYAML string, instrumentationConfigYAML string) (*model.K8sActualSource, error) {
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
		Namespace:                 pw.Namespace,
		Kind:                      kindToGql(string(pw.Kind)),
		Name:                      pw.Name,
		Selected:                  &selected,
		DataStreamNames:           dataStreamNames,
		OtelServiceName:           &instruConfig.Spec.ServiceName,
		NumberOfInstances:         nil,
		Containers:                containers,
		Conditions:                services.ConvertConditions(instruConfig.Status.Conditions),
		ManifestYaml:              &manifestYAML,
		InstrumentationConfigYaml: &instrumentationConfigYAML,
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

const defaultProvenanceSource = "odigos-configuration"

func provenanceFor(prov map[string]string, path string) string {
	if prov != nil {
		if s, ok := prov[path]; ok {
			return s
		}
	}
	return defaultProvenanceSource
}

func reconciledBool(val bool, path string, prov map[string]string) *model.ReconciledBoolean {
	return &model.ReconciledBoolean{ReconciledFrom: provenanceFor(prov, path), Value: &val}
}

func reconciledBoolPtr(val *bool, path string, prov map[string]string) *model.ReconciledBoolean {
	if val == nil {
		return nil
	}
	return &model.ReconciledBoolean{ReconciledFrom: provenanceFor(prov, path), Value: val}
}

func reconciledStr(val string, path string, prov map[string]string) *model.ReconciledString {
	if val == "" {
		return nil
	}
	return &model.ReconciledString{ReconciledFrom: provenanceFor(prov, path), Value: &val}
}

func reconciledStrAlways(val string, path string, prov map[string]string) *model.ReconciledString {
	return &model.ReconciledString{ReconciledFrom: provenanceFor(prov, path), Value: &val}
}

func reconciledStrPtr(val *string, path string, prov map[string]string) *model.ReconciledString {
	if val == nil {
		return nil
	}
	return &model.ReconciledString{ReconciledFrom: provenanceFor(prov, path), Value: val}
}

func reconciledIntNonZero(val int, path string, prov map[string]string) *model.ReconciledInt {
	if val == 0 {
		return nil
	}
	return &model.ReconciledInt{ReconciledFrom: provenanceFor(prov, path), Value: &val}
}

func reconciledStrArray(val []string, path string, prov map[string]string) *model.ReconciledStringArray {
	return &model.ReconciledStringArray{ReconciledFrom: provenanceFor(prov, path), Value: val}
}

func EffectiveConfigToModel(config *common.OdigosConfiguration, prov map[string]string) (*model.EffectiveConfig, error) {
	if config == nil {
		return nil, nil
	}

	result := &model.EffectiveConfig{
		ConfigVersion: config.ConfigVersion,
	}

	// Non-pointer booleans (always present)
	result.TelemetryEnabled = reconciledBool(config.TelemetryEnabled, "telemetryEnabled", prov)
	result.OpenshiftEnabled = reconciledBool(config.OpenshiftEnabled, "openshiftEnabled", prov)
	result.Psp = reconciledBool(config.Psp, "psp", prov)
	result.SkipWebhookIssuerCreation = reconciledBool(config.SkipWebhookIssuerCreation, "skipWebhookIssuerCreation", prov)

	// Pointer booleans (nil when unset)
	result.IgnoreOdigosNamespace = reconciledBoolPtr(config.IgnoreOdigosNamespace, "ignoreOdigosNamespace", prov)
	result.AllowConcurrentAgents = reconciledBoolPtr(config.AllowConcurrentAgents, "allowConcurrentAgents", prov)
	result.KarpenterEnabled = reconciledBoolPtr(config.KarpenterEnabled, "karpenterEnabled", prov)
	result.RollbackDisabled = reconciledBoolPtr(config.RollbackDisabled, "rollbackDisabled", prov)
	result.ClickhouseJSONTypeEnabled = reconciledBoolPtr(config.ClickhouseJsonTypeEnabledProperty, "clickhouseJsonTypeEnabled", prov)
	result.CheckDeviceHealthBeforeInjection = reconciledBoolPtr(config.CheckDeviceHealthBeforeInjection, "checkDeviceHealthBeforeInjection", prov)
	result.WaspEnabled = reconciledBoolPtr(config.WaspEnabled, "waspEnabled", prov)

	// Strings (nil when empty)
	result.ImagePrefix = reconciledStr(config.ImagePrefix, "imagePrefix", prov)
	result.UIRemoteURL = reconciledStr(config.UiRemoteUrl, "uiRemoteUrl", prov)
	result.CentralBackendURL = reconciledStr(config.CentralBackendURL, "centralBackendURL", prov)
	result.ClusterName = reconciledStr(config.ClusterName, "clusterName", prov)
	result.CustomContainerRuntimeSocketPath = reconciledStr(config.CustomContainerRuntimeSocketPath, "customContainerRuntimeSocketPath", prov)
	result.RollbackGraceTime = reconciledStr(config.RollbackGraceTime, "rollbackGraceTime", prov)
	result.RollbackStabilityWindow = reconciledStr(config.RollbackStabilityWindow, "rollbackStabilityWindow", prov)
	result.GoAutoOffsetsCron = reconciledStr(config.GoAutoOffsetsCron, "goAutoOffsetsCron", prov)
	result.GoAutoOffsetsMode = reconciledStr(config.GoAutoOffsetsMode, "goAutoOffsetsMode", prov)
	result.ResourceSizePreset = reconciledStr(config.ResourceSizePreset, "resourceSizePreset", prov)
	result.TraceIDSuffix = reconciledStr(config.TraceIdSuffix, "traceIdSuffix", prov)

	// Ints (nil when zero)
	result.UIPaginationLimit = reconciledIntNonZero(config.UiPaginationLimit, "uiPaginationLimit", prov)
	result.OdigletHealthProbeBindPort = reconciledIntNonZero(config.OdigletHealthProbeBindPort, "odigletHealthProbeBindPort", prov)

	// Arrays
	result.IgnoredNamespaces = reconciledStrArray(config.IgnoredNamespaces, "ignoredNamespaces", prov)
	result.IgnoredContainers = reconciledStrArray(config.IgnoredContainers, "ignoredContainers", prov)
	result.AllowedTestConnectionHosts = reconciledStrArray(config.AllowedTestConnectionHosts, "allowedTestConnectionHosts", prov)
	result.ImagePullSecrets = reconciledStrArray(config.ImagePullSecrets, "imagePullSecrets", prov)

	if len(config.Profiles) > 0 {
		profiles := make([]string, len(config.Profiles))
		for i, p := range config.Profiles {
			profiles[i] = string(p)
		}
		result.Profiles = reconciledStrArray(profiles, "profiles", prov)
	} else {
		result.Profiles = reconciledStrArray(nil, "profiles", prov)
	}

	// Enums
	if config.UiMode != "" {
		uiMode := convertUiModeToModel(config.UiMode)
		result.UIMode = &model.ReconciledUIMode{ReconciledFrom: provenanceFor(prov, "uiMode"), Value: &uiMode}
	}
	if config.MountMethod != nil {
		mountMethod := convertMountMethodToModel(*config.MountMethod)
		result.MountMethod = &model.ReconciledMountMethod{ReconciledFrom: provenanceFor(prov, "mountMethod"), Value: &mountMethod}
	}
	if config.AgentEnvVarsInjectionMethod != nil {
		injMethod := convertEnvInjectionMethodToModel(*config.AgentEnvVarsInjectionMethod)
		result.AgentEnvVarsInjectionMethod = &model.ReconciledEnvInjectionMethod{ReconciledFrom: provenanceFor(prov, "agentEnvVarsInjectionMethod"), Value: &injMethod}
	}

	// Component log levels
	setEffectiveConfigComponentLogLevels(result, config, prov)

	// Nested structs
	if err := setEffectiveConfigNestedStructs(result, config, prov); err != nil {
		return nil, err
	}

	return result, nil
}

func setEffectiveConfigComponentLogLevels(result *model.EffectiveConfig, config *common.OdigosConfiguration, prov map[string]string) {
	out := &model.ComponentLogLevelsConfig{}
	resolve := func(component string) model.OdigosLogLevel {
		if config.ComponentLogLevels == nil {
			return model.OdigosLogLevel(common.LogLevelInfo)
		}
		return model.OdigosLogLevel(config.ComponentLogLevels.Resolve(component))
	}
	reconciledLogLevel := func(component, path string) *model.ReconciledOdigosLogLevel {
		lvl := resolve(component)
		return &model.ReconciledOdigosLogLevel{ReconciledFrom: provenanceFor(prov, path), Value: &lvl}
	}
	out.Default = reconciledLogLevel("default", "componentLogLevels.default")
	out.Autoscaler = reconciledLogLevel("autoscaler", "componentLogLevels.autoscaler")
	out.Scheduler = reconciledLogLevel("scheduler", "componentLogLevels.scheduler")
	out.Instrumentor = reconciledLogLevel("instrumentor", "componentLogLevels.instrumentor")
	out.Odiglet = reconciledLogLevel("odiglet", "componentLogLevels.odiglet")
	out.Deviceplugin = reconciledLogLevel("deviceplugin", "componentLogLevels.deviceplugin")
	out.UI = reconciledLogLevel("ui", "componentLogLevels.ui")
	out.Collector = reconciledLogLevel("collector", "componentLogLevels.collector")
	result.ComponentLogLevels = out
}

func setEffectiveConfigNestedStructs(result *model.EffectiveConfig, config *common.OdigosConfiguration, prov map[string]string) error {
	if len(config.NodeSelector) > 0 {
		nodeSelectorJSON, err := json.Marshal(config.NodeSelector)
		if err != nil {
			return fmt.Errorf("failed to marshal nodeSelector: %w", err)
		}
		nodeSelectorStr := string(nodeSelectorJSON)
		result.NodeSelector = &model.ReconciledString{ReconciledFrom: provenanceFor(prov, "nodeSelector"), Value: &nodeSelectorStr}
	}

	if config.CollectorGateway != nil {
		collectorGateway, err := convertCollectorGatewayToModel(config.CollectorGateway, prov)
		if err != nil {
			return err
		}
		result.CollectorGateway = collectorGateway
	}

	if config.CollectorNode != nil {
		result.CollectorNode = convertCollectorNodeToModel(config.CollectorNode, prov)
	}

	if config.Rollout != nil {
		result.Rollout = &model.RolloutConfig{
			AutomaticRolloutDisabled: reconciledBoolPtr(config.Rollout.AutomaticRolloutDisabled, "rollout.automaticRolloutDisabled", prov),
		}
	}

	if config.Oidc != nil {
		result.Oidc = convertOidcToModel(config.Oidc, prov)
	}

	if config.UserInstrumentationEnvs != nil {
		userInstrumentationEnvs, err := convertUserInstrumentationEnvsToModel(config.UserInstrumentationEnvs, prov)
		if err != nil {
			return err
		}
		result.UserInstrumentationEnvs = userInstrumentationEnvs
	}

	if config.MetricsSources != nil {
		result.MetricsSources = convertMetricsSourcesToModel(config.MetricsSources, prov)
	}

	if config.AgentsInitContainerResources != nil {
		result.AgentsInitContainerResources = convertAgentsInitContainerResourcesToModel(config.AgentsInitContainerResources, prov)
	}

	if config.OdigosOwnTelemetryStore != nil {
		result.OdigosOwnTelemetryStore = &model.OdigosOwnTelemetryConfig{
			MetricsStoreDisabled: reconciledBoolPtr(config.OdigosOwnTelemetryStore.MetricsStoreDisabled, "odigosOwnTelemetryStore.metricsStoreDisabled", prov),
		}
	}

	return nil
}

func convertOidcToModel(oidc *common.OidcConfiguration, prov map[string]string) *model.OidcConfig {
	return &model.OidcConfig{
		TenantURL:    reconciledStr(oidc.TenantUrl, "oidc.tenantUrl", prov),
		ClientID:     reconciledStr(oidc.ClientId, "oidc.clientId", prov),
		ClientSecret: reconciledStr(oidc.ClientSecret, "oidc.clientSecret", prov),
	}
}

func convertAgentsInitContainerResourcesToModel(resources *common.AgentsInitContainerResources, prov map[string]string) *model.AgentsInitContainerResourcesConfig {
	return &model.AgentsInitContainerResourcesConfig{
		RequestCPUm:      reconciledIntNonZero(resources.RequestCPUm, "agentsInitContainerResources.requestCPUm", prov),
		LimitCPUm:        reconciledIntNonZero(resources.LimitCPUm, "agentsInitContainerResources.limitCPUm", prov),
		RequestMemoryMiB: reconciledIntNonZero(resources.RequestMemoryMiB, "agentsInitContainerResources.requestMemoryMiB", prov),
		LimitMemoryMiB:   reconciledIntNonZero(resources.LimitMemoryMiB, "agentsInitContainerResources.limitMemoryMiB", prov),
	}
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

func convertCollectorGatewayToModel(gw *common.CollectorGatewayConfiguration, prov map[string]string) (*model.CollectorGatewayConfig, error) {
	if gw == nil {
		return nil, nil
	}

	p := func(field string) string { return "collectorGateway." + field }
	result := &model.CollectorGatewayConfig{
		MinReplicas:                reconciledIntNonZero(gw.MinReplicas, p("minReplicas"), prov),
		MaxReplicas:                reconciledIntNonZero(gw.MaxReplicas, p("maxReplicas"), prov),
		RequestMemoryMiB:           reconciledIntNonZero(gw.RequestMemoryMiB, p("requestMemoryMiB"), prov),
		LimitMemoryMiB:             reconciledIntNonZero(gw.LimitMemoryMiB, p("limitMemoryMiB"), prov),
		RequestCPUm:                reconciledIntNonZero(gw.RequestCPUm, p("requestCPUm"), prov),
		LimitCPUm:                  reconciledIntNonZero(gw.LimitCPUm, p("limitCPUm"), prov),
		MemoryLimiterLimitMiB:      reconciledIntNonZero(gw.MemoryLimiterLimitMiB, p("memoryLimiterLimitMiB"), prov),
		MemoryLimiterSpikeLimitMiB: reconciledIntNonZero(gw.MemoryLimiterSpikeLimitMiB, p("memoryLimiterSpikeLimitMiB"), prov),
		GoMemLimitMiB:              reconciledIntNonZero(gw.GoMemLimitMib, p("goMemLimitMiB"), prov),
		ClusterMetricsEnabled:      reconciledBoolPtr(gw.ClusterMetricsEnabled, p("clusterMetricsEnabled"), prov),
		HTTPSProxyAddress:          reconciledStrPtr(gw.HttpsProxyAddress, p("httpsProxyAddress"), prov),
	}

	if gw.ServiceGraph != nil {
		result.ServiceGraphDisabled = reconciledBoolPtr(gw.ServiceGraph.Disabled, p("serviceGraphDisabled"), prov)
	}

	if gw.NodeSelector != nil && len(*gw.NodeSelector) > 0 {
		nodeSelectorJSON, err := json.Marshal(*gw.NodeSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal collectorGateway.nodeSelector: %w", err)
		}
		nodeSelectorStr := string(nodeSelectorJSON)
		result.NodeSelector = &model.ReconciledString{ReconciledFrom: provenanceFor(prov, p("nodeSelector")), Value: &nodeSelectorStr}
	}

	return result, nil
}

func convertCollectorNodeToModel(node *common.CollectorNodeConfiguration, prov map[string]string) *model.CollectorNodeConfig {
	if node == nil {
		return nil
	}

	p := func(field string) string { return "collectorNode." + field }
	result := &model.CollectorNodeConfig{
		CollectorOwnMetricsPort:    reconciledIntNonZero(int(node.CollectorOwnMetricsPort), p("collectorOwnMetricsPort"), prov),
		RequestMemoryMiB:           reconciledIntNonZero(node.RequestMemoryMiB, p("requestMemoryMiB"), prov),
		LimitMemoryMiB:             reconciledIntNonZero(node.LimitMemoryMiB, p("limitMemoryMiB"), prov),
		RequestCPUm:                reconciledIntNonZero(node.RequestCPUm, p("requestCPUm"), prov),
		LimitCPUm:                  reconciledIntNonZero(node.LimitCPUm, p("limitCPUm"), prov),
		MemoryLimiterLimitMiB:      reconciledIntNonZero(node.MemoryLimiterLimitMiB, p("memoryLimiterLimitMiB"), prov),
		MemoryLimiterSpikeLimitMiB: reconciledIntNonZero(node.MemoryLimiterSpikeLimitMiB, p("memoryLimiterSpikeLimitMiB"), prov),
		GoMemLimitMiB:              reconciledIntNonZero(node.GoMemLimitMib, p("goMemLimitMiB"), prov),
		EnableDataCompression:      reconciledBoolPtr(node.EnableDataCompression, p("enableDataCompression"), prov),
	}

	if node.OtlpExporterConfiguration != nil {
		result.OtlpExporterConfiguration = convertOtlpExporterToModel(node.OtlpExporterConfiguration, prov)
	}

	return result
}

func convertOtlpExporterToModel(otlp *common.OtlpExporterConfiguration, prov map[string]string) *model.OtlpExporterConfig {
	if otlp == nil {
		return nil
	}

	p := func(field string) string { return "collectorNode.otlpExporterConfiguration." + field }
	result := &model.OtlpExporterConfig{
		EnableDataCompression: reconciledBoolPtr(otlp.EnableDataCompression, p("enableDataCompression"), prov),
		Timeout:               reconciledStr(otlp.Timeout, p("timeout"), prov),
	}

	if otlp.RetryOnFailure != nil {
		rp := func(field string) string { return p("retryOnFailure." + field) }
		result.RetryOnFailure = &model.RetryOnFailureConfig{
			Enabled:         reconciledBoolPtr(otlp.RetryOnFailure.Enabled, rp("enabled"), prov),
			InitialInterval: reconciledStr(otlp.RetryOnFailure.InitialInterval, rp("initialInterval"), prov),
			MaxInterval:     reconciledStr(otlp.RetryOnFailure.MaxInterval, rp("maxInterval"), prov),
			MaxElapsedTime:  reconciledStr(otlp.RetryOnFailure.MaxElapsedTime, rp("maxElapsedTime"), prov),
		}
	}

	return result
}

func convertUserInstrumentationEnvsToModel(envs *common.UserInstrumentationEnvs, prov map[string]string) (*model.UserInstrumentationEnvsConfig, error) {
	if envs == nil {
		return nil, nil
	}

	result := &model.UserInstrumentationEnvsConfig{}

	if len(envs.Languages) > 0 {
		languagesJSON, err := json.Marshal(envs.Languages)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal userInstrumentationEnvs.languages: %w", err)
		}
		languagesStr := string(languagesJSON)
		result.Languages = &model.ReconciledString{ReconciledFrom: provenanceFor(prov, "userInstrumentationEnvs.languages"), Value: &languagesStr}
	}

	return result, nil
}

// convertOdigosConfigToSamplingConfig converts common.OdigosConfiguration (or its Sampling slice) to the GraphQL model.SamplingConfig.
// Placed here so all common → graph model conversions live in one place.
func convertOdigosConfigToSamplingConfig(config *common.OdigosConfiguration) *model.SamplingConfig {
	if config == nil || config.Sampling == nil {
		return nil
	}
	s := config.Sampling
	out := &model.SamplingConfig{}
	if s.TailSampling != nil {
		out.TailSampling = &model.TailSamplingConfig{
			Disabled:                     s.TailSampling.Disabled,
			TraceAggregationWaitDuration: s.TailSampling.TraceAggregationWaitDuration,
		}
	}
	if s.K8sHealthProbesSampling != nil {
		out.K8sHealthProbesSampling = &model.K8sHealthProbesSamplingConfig{
			Enabled:        s.K8sHealthProbesSampling.Enabled,
			KeepPercentage: s.K8sHealthProbesSampling.KeepPercentage,
		}
	}
	return out
}

func convertSamplingConfigInputToOdigosConfig(config *model.SamplingConfigInput) *common.SamplingConfiguration {
	if config == nil {
		return nil
	}
	result := &common.SamplingConfiguration{}
	if config.TailSampling != nil {
		result.TailSampling = &sampling.TailSamplingConfiguration{
			Disabled:                     config.TailSampling.Disabled,
			TraceAggregationWaitDuration: config.TailSampling.TraceAggregationWaitDuration,
		}
	}
	if config.K8sHealthProbesSampling != nil {
		result.K8sHealthProbesSampling = &common.K8sHealthProbesSamplingConfiguration{
			Enabled:        config.K8sHealthProbesSampling.Enabled,
			KeepPercentage: config.K8sHealthProbesSampling.KeepPercentage,
		}
	}
	return result
}

// headSamplingOperatorToModel maps v1alpha1.Operator to the GraphQL head sampling condition operator enum.
func headSamplingOperatorToModel(op v1alpha1.Operator) model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckConditionOperator {
	switch op {
	case v1alpha1.Equals:
		return model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckConditionOperatorEquals
	case v1alpha1.NotEquals:
		return model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckConditionOperatorNotEquals
	case v1alpha1.EndWith:
		return model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckConditionOperatorEndWith
	case v1alpha1.StartWith:
		return model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckConditionOperatorStartWith
	default:
		return model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckConditionOperatorEquals
	}
}

// containerAgentConfigToAgentConfigModel converts InstrumentationConfig container agent config (traces/head sampling) to the GraphQL K8sWorkloadContainerAgentConfig model.
func containerAgentConfigToAgentConfigModel(c *v1alpha1.ContainerAgentConfig) *model.K8sWorkloadContainerAgentConfig {
	if c == nil || c.Traces == nil || c.Traces.HeadSampling == nil {
		return nil
	}
	hs := c.Traces.HeadSampling
	checks := make([]*model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheck, 0, len(hs.AttributesAndSamplerRules))
	for i := range hs.AttributesAndSamplerRules {
		rule := &hs.AttributesAndSamplerRules[i]
		conditions := make([]*model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckCondition, 0, len(rule.AttributeConditions))
		for j := range rule.AttributeConditions {
			ac := &rule.AttributeConditions[j]
			conditions = append(conditions, &model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheckCondition{
				Key:      ac.Key,
				Operator: headSamplingOperatorToModel(ac.Operator),
				Value:    ac.Val,
			})
		}
		checks = append(checks, &model.K8sWorkloadContainerAgentConfigTracesHeadSamplingCheck{
			Conditions: conditions,
			Percentage: rule.Fraction * 100,
		})
	}
	return &model.K8sWorkloadContainerAgentConfig{
		Traces: &model.K8sWorkloadContainerAgentConfigTraces{
			HeadSampling: &model.K8sWorkloadContainerAgentConfigTracesHeadSampling{
				Checks:             checks,
				FallbackPercentage: hs.FallbackFraction * 100,
			},
		},
	}
}

func convertMetricsSourcesToModel(ms *common.MetricsSourceConfiguration, prov map[string]string) *model.MetricsSourceConfig {
	if ms == nil {
		return nil
	}

	result := &model.MetricsSourceConfig{}

	if ms.SpanMetrics != nil {
		sm := ms.SpanMetrics
		p := func(f string) string { return "metricsSources.spanMetrics." + f }
		result.SpanMetrics = &model.MetricsSourceSpanMetricsConfig{
			Disabled:                     reconciledBoolPtr(sm.Disabled, p("disabled"), prov),
			Interval:                     reconciledStr(sm.Interval, p("interval"), prov),
			MetricsExpiration:            reconciledStr(sm.MetricsExpiration, p("metricsExpiration"), prov),
			AdditionalDimensions:         reconciledStrArray(sm.AdditionalDimensions, p("additionalDimensions"), prov),
			HistogramBuckets:             reconciledStrArray(sm.ExplicitHistogramBuckets, p("histogramBuckets"), prov),
			IncludedProcessInDimensions:  reconciledBoolPtr(sm.IncludedProcessInDimensions, p("includedProcessInDimensions"), prov),
			ExcludedResourceAttributes:   reconciledStrArray(sm.ExcludedResourceAttributes, p("excludedResourceAttributes"), prov),
			ResourceMetricsKeyAttributes: reconciledStrArray(sm.ResourceMetricsKeyAttributes, p("resourceMetricsKeyAttributes"), prov),
		}
		if sm.HistogramDisabled {
			result.SpanMetrics.HistogramDisabled = reconciledBool(sm.HistogramDisabled, p("histogramDisabled"), prov)
		}
	}

	if ms.HostMetrics != nil {
		p := func(f string) string { return "metricsSources.hostMetrics." + f }
		result.HostMetrics = &model.MetricsSourceHostMetricsConfig{
			Disabled: reconciledBoolPtr(ms.HostMetrics.Disabled, p("disabled"), prov),
			Interval: reconciledStr(ms.HostMetrics.Interval, p("interval"), prov),
		}
	}

	if ms.KubeletStats != nil {
		p := func(f string) string { return "metricsSources.kubeletStats." + f }
		result.KubeletStats = &model.MetricsSourceKubeletStatsConfig{
			Disabled: reconciledBoolPtr(ms.KubeletStats.Disabled, p("disabled"), prov),
			Interval: reconciledStr(ms.KubeletStats.Interval, p("interval"), prov),
		}
	}

	if ms.OdigosOwnMetrics != nil {
		result.OdigosOwnMetrics = &model.MetricsSourceOdigosOwnMetricsConfig{
			Interval: reconciledStr(ms.OdigosOwnMetrics.Interval, "metricsSources.odigosOwnMetrics.interval", prov),
		}
	}

	if ms.AgentMetrics != nil {
		result.AgentMetrics = &model.MetricsSourceAgentMetricsConfig{}

		if ms.AgentMetrics.SpanMetrics != nil {
			result.AgentMetrics.SpanMetrics = &model.MetricsSourceAgentSpanMetricsConfig{
				Enabled: reconciledBool(ms.AgentMetrics.SpanMetrics.Enabled, "metricsSources.agentMetrics.spanMetrics.enabled", prov),
			}
		}

		if ms.AgentMetrics.RuntimeMetrics != nil && ms.AgentMetrics.RuntimeMetrics.Java != nil {
			result.AgentMetrics.RuntimeMetrics = &model.MetricsSourceAgentRuntimeMetricsConfig{
				Java: &model.MetricsSourceAgentJavaRuntimeMetricsConfig{
					Disabled: reconciledBoolPtr(ms.AgentMetrics.RuntimeMetrics.Java.Disabled, "metricsSources.agentMetrics.runtimeMetrics.java.disabled", prov),
				},
			}

			if len(ms.AgentMetrics.RuntimeMetrics.Java.Metrics) > 0 {
				metrics := make([]*model.MetricsSourceAgentRuntimeMetricConfig, len(ms.AgentMetrics.RuntimeMetrics.Java.Metrics))
				for i, m := range ms.AgentMetrics.RuntimeMetrics.Java.Metrics {
					basePath := fmt.Sprintf("metricsSources.agentMetrics.runtimeMetrics.java.metrics.%d", i)
					metrics[i] = &model.MetricsSourceAgentRuntimeMetricConfig{
						Name:     reconciledStrAlways(m.Name, basePath+".name", prov),
						Disabled: reconciledBoolPtr(m.Disabled, basePath+".disabled", prov),
					}
				}
				result.AgentMetrics.RuntimeMetrics.Java.Metrics = metrics
			}
		}
	}

	return result
}
