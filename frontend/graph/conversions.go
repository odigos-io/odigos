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

type provenanceCollector struct {
	prov    map[string]string
	entries []*model.ProvenanceEntry
}

func newProvenanceCollector(prov map[string]string) *provenanceCollector {
	return &provenanceCollector{prov: prov}
}

func (pc *provenanceCollector) record(path string) {
	pc.entries = append(pc.entries, &model.ProvenanceEntry{
		HelmPath:       path,
		ReconciledFrom: provenanceFor(pc.prov, path),
	})
}

// recordAs looks up provenance using the YAML config key but records the helm value path.
func (pc *provenanceCollector) recordAs(yamlKey, helmPath string) {
	pc.entries = append(pc.entries, &model.ProvenanceEntry{
		HelmPath:       helmPath,
		ReconciledFrom: provenanceFor(pc.prov, yamlKey),
	})
}

func ptrBool(v bool) *bool    { return &v }
func ptrStr(v string) *string { return &v }
func ptrInt(v int) *int       { return &v }

func EffectiveConfigToModel(config *common.OdigosConfiguration, prov map[string]string) (*model.EffectiveConfig, error) {
	if config == nil {
		return nil, nil
	}

	pc := newProvenanceCollector(prov)

	result := &model.EffectiveConfig{
		ConfigVersion: config.ConfigVersion,
	}

	// Non-pointer booleans (always present)
	result.TelemetryEnabled = ptrBool(config.TelemetryEnabled)
	pc.record("telemetryEnabled")
	result.OpenshiftEnabled = ptrBool(config.OpenshiftEnabled)
	pc.record("openshiftEnabled")
	result.Psp = ptrBool(config.Psp)
	pc.record("psp")
	result.SkipWebhookIssuerCreation = ptrBool(config.SkipWebhookIssuerCreation)
	pc.record("skipWebhookIssuerCreation")

	// Pointer booleans (nil when unset)
	if config.IgnoreOdigosNamespace != nil {
		result.IgnoreOdigosNamespace = config.IgnoreOdigosNamespace
		pc.record("ignoreOdigosNamespace")
	}
	if config.ClickhouseJsonTypeEnabledProperty != nil {
		result.ClickhouseJSONTypeEnabled = config.ClickhouseJsonTypeEnabledProperty
		pc.record("clickhouseJsonTypeEnabled")
	}

	// Nested wrapper types that match LocalUiConfigInput structure / helm value paths
	result.AllowConcurrentAgents = &model.AllowConcurrentAgentsConfig{}
	if config.AllowConcurrentAgents != nil {
		result.AllowConcurrentAgents.Enabled = config.AllowConcurrentAgents
		pc.recordAs("allowConcurrentAgents", "allowConcurrentAgents.enabled")
	}

	result.Karpenter = &model.KarpenterConfig{}
	if config.KarpenterEnabled != nil {
		result.Karpenter.Enabled = config.KarpenterEnabled
		pc.recordAs("karpenterEnabled", "karpenter.enabled")
	}

	result.Wasp = &model.WaspConfig{}
	if config.WaspEnabled != nil {
		result.Wasp.Enabled = config.WaspEnabled
		pc.recordAs("waspEnabled", "wasp.enabled")
	}

	result.Instrumentor = &model.InstrumentorConfig{}
	if config.CheckDeviceHealthBeforeInjection != nil {
		result.Instrumentor.CheckDeviceHealthBeforeInjection = config.CheckDeviceHealthBeforeInjection
		pc.recordAs("checkDeviceHealthBeforeInjection", "instrumentor.checkDeviceHealthBeforeInjection")
	}

	// Strings (nil when empty)
	if config.ImagePrefix != "" {
		result.ImagePrefix = ptrStr(config.ImagePrefix)
		pc.record("imagePrefix")
	}
	if config.UiRemoteUrl != "" {
		result.UIRemoteURL = ptrStr(config.UiRemoteUrl)
		pc.record("uiRemoteUrl")
	}
	if config.CentralBackendURL != "" {
		result.CentralBackendURL = ptrStr(config.CentralBackendURL)
		pc.record("centralBackendURL")
	}
	if config.ClusterName != "" {
		result.ClusterName = ptrStr(config.ClusterName)
		pc.record("clusterName")
	}
	if config.CustomContainerRuntimeSocketPath != "" {
		result.CustomContainerRuntimeSocketPath = ptrStr(config.CustomContainerRuntimeSocketPath)
		pc.record("customContainerRuntimeSocketPath")
	}
	if config.GoAutoOffsetsCron != "" {
		result.GoAutoOffsetsCron = ptrStr(config.GoAutoOffsetsCron)
		pc.record("goAutoOffsetsCron")
	}
	if config.GoAutoOffsetsMode != "" {
		result.GoAutoOffsetsMode = ptrStr(config.GoAutoOffsetsMode)
		pc.record("goAutoOffsetsMode")
	}
	if config.ResourceSizePreset != "" {
		result.ResourceSizePreset = ptrStr(config.ResourceSizePreset)
		pc.record("resourceSizePreset")
	}
	if config.TraceIdSuffix != "" {
		result.TraceIDSuffix = ptrStr(config.TraceIdSuffix)
		pc.record("traceIdSuffix")
	}

	// Ints (nil when zero)
	if config.UiPaginationLimit != 0 {
		result.UIPaginationLimit = ptrInt(config.UiPaginationLimit)
		pc.record("uiPaginationLimit")
	}
	if config.OdigletHealthProbeBindPort != 0 {
		result.OdigletHealthProbeBindPort = ptrInt(config.OdigletHealthProbeBindPort)
		pc.record("odigletHealthProbeBindPort")
	}

	// Arrays
	result.IgnoredNamespaces = config.IgnoredNamespaces
	pc.record("ignoredNamespaces")
	result.IgnoredContainers = config.IgnoredContainers
	pc.record("ignoredContainers")
	result.AllowedTestConnectionHosts = config.AllowedTestConnectionHosts
	pc.record("allowedTestConnectionHosts")
	result.ImagePullSecrets = config.ImagePullSecrets
	pc.record("imagePullSecrets")

	if len(config.Profiles) > 0 {
		profiles := make([]string, len(config.Profiles))
		for i, p := range config.Profiles {
			profiles[i] = string(p)
		}
		result.Profiles = profiles
	}
	pc.record("profiles")

	// Enums
	if config.UiMode != "" {
		uiMode := convertUiModeToModel(config.UiMode)
		result.UIMode = &uiMode
		pc.record("uiMode")
	}
	if config.MountMethod != nil {
		mountMethod := convertMountMethodToModel(*config.MountMethod)
		result.Instrumentor.MountMethod = &mountMethod
		pc.recordAs("mountMethod", "instrumentor.mountMethod")
	}
	if config.AgentEnvVarsInjectionMethod != nil {
		injMethod := convertEnvInjectionMethodToModel(*config.AgentEnvVarsInjectionMethod)
		result.Instrumentor.AgentEnvVarsInjectionMethod = &injMethod
		pc.recordAs("agentEnvVarsInjectionMethod", "instrumentor.agentEnvVarsInjectionMethod")
	}

	// Component log levels
	setEffectiveConfigComponentLogLevels(result, config, pc)

	// Nested structs
	if err := setEffectiveConfigNestedStructs(result, config, pc); err != nil {
		return nil, err
	}

	result.Provenance = pc.entries

	return result, nil
}

func setEffectiveConfigComponentLogLevels(result *model.EffectiveConfig, config *common.OdigosConfiguration, pc *provenanceCollector) {
	out := &model.ComponentLogLevelsConfig{}
	resolve := func(component string) model.OdigosLogLevel {
		if config.ComponentLogLevels == nil {
			return model.OdigosLogLevel(common.LogLevelInfo)
		}
		return model.OdigosLogLevel(config.ComponentLogLevels.Resolve(component))
	}
	setLogLevel := func(component, path string) *model.OdigosLogLevel {
		lvl := resolve(component)
		pc.record(path)
		return &lvl
	}
	out.Default = setLogLevel("default", "componentLogLevels.default")
	out.Autoscaler = setLogLevel("autoscaler", "componentLogLevels.autoscaler")
	out.Scheduler = setLogLevel("scheduler", "componentLogLevels.scheduler")
	out.Instrumentor = setLogLevel("instrumentor", "componentLogLevels.instrumentor")
	out.Odiglet = setLogLevel("odiglet", "componentLogLevels.odiglet")
	out.Deviceplugin = setLogLevel("deviceplugin", "componentLogLevels.deviceplugin")
	out.UI = setLogLevel("ui", "componentLogLevels.ui")
	out.Collector = setLogLevel("collector", "componentLogLevels.collector")
	result.ComponentLogLevels = out
}

func setEffectiveConfigNestedStructs(result *model.EffectiveConfig, config *common.OdigosConfiguration, pc *provenanceCollector) error {
	if len(config.NodeSelector) > 0 {
		nodeSelectorJSON, err := json.Marshal(config.NodeSelector)
		if err != nil {
			return fmt.Errorf("failed to marshal nodeSelector: %w", err)
		}
		nodeSelectorStr := string(nodeSelectorJSON)
		result.NodeSelector = &nodeSelectorStr
		pc.record("nodeSelector")
	}

	if config.CollectorGateway != nil {
		collectorGateway, err := convertCollectorGatewayToModel(config.CollectorGateway, pc)
		if err != nil {
			return err
		}
		result.CollectorGateway = collectorGateway
	}

	if config.CollectorNode != nil {
		result.CollectorNode = convertCollectorNodeToModel(config.CollectorNode, pc)
	}

	if config.Rollout != nil {
		result.Rollout = &model.RolloutConfig{
			AutomaticRolloutDisabled: config.Rollout.AutomaticRolloutDisabled,
		}
		pc.record("rollout.automaticRolloutDisabled")
	}

	result.AutoRollback = &model.AutoRollbackConfig{}
	if config.RollbackDisabled != nil {
		result.AutoRollback.Disabled = config.RollbackDisabled
		pc.recordAs("rollbackDisabled", "autoRollback.disabled")
	}
	if config.RollbackGraceTime != "" {
		result.AutoRollback.GraceTime = ptrStr(config.RollbackGraceTime)
		pc.recordAs("rollbackGraceTime", "autoRollback.graceTime")
	}
	if config.RollbackStabilityWindow != "" {
		result.AutoRollback.StabilityWindowTime = ptrStr(config.RollbackStabilityWindow)
		pc.recordAs("rollbackStabilityWindow", "autoRollback.stabilityWindowTime")
	}

	if config.Oidc != nil {
		result.Oidc = convertOidcToModel(config.Oidc, pc)
	}

	if config.UserInstrumentationEnvs != nil {
		userInstrumentationEnvs, err := convertUserInstrumentationEnvsToModel(config.UserInstrumentationEnvs, pc)
		if err != nil {
			return err
		}
		result.UserInstrumentationEnvs = userInstrumentationEnvs
	}

	if config.MetricsSources != nil {
		result.MetricsSources = convertMetricsSourcesToModel(config.MetricsSources, pc)
	}

	if config.AgentsInitContainerResources != nil {
		result.AgentsInitContainerResources = convertAgentsInitContainerResourcesToModel(config.AgentsInitContainerResources, pc)
	}

	if config.OdigosOwnTelemetryStore != nil {
		result.OdigosOwnTelemetryStore = &model.OdigosOwnTelemetryConfig{
			MetricsStoreDisabled: config.OdigosOwnTelemetryStore.MetricsStoreDisabled,
		}
		pc.record("odigosOwnTelemetryStore.metricsStoreDisabled")
	}

	return nil
}

func convertOidcToModel(oidc *common.OidcConfiguration, pc *provenanceCollector) *model.OidcConfig {
	result := &model.OidcConfig{}
	if oidc.TenantUrl != "" {
		result.TenantURL = ptrStr(oidc.TenantUrl)
		pc.record("oidc.tenantUrl")
	}
	if oidc.ClientId != "" {
		result.ClientID = ptrStr(oidc.ClientId)
		pc.record("oidc.clientId")
	}
	if oidc.ClientSecret != "" {
		result.ClientSecret = ptrStr(oidc.ClientSecret)
		pc.record("oidc.clientSecret")
	}
	return result
}

func convertAgentsInitContainerResourcesToModel(resources *common.AgentsInitContainerResources, pc *provenanceCollector) *model.AgentsInitContainerResourcesConfig {
	result := &model.AgentsInitContainerResourcesConfig{}
	if resources.RequestCPUm != 0 {
		result.RequestCPUm = ptrInt(resources.RequestCPUm)
		pc.record("agentsInitContainerResources.requestCPUm")
	}
	if resources.LimitCPUm != 0 {
		result.LimitCPUm = ptrInt(resources.LimitCPUm)
		pc.record("agentsInitContainerResources.limitCPUm")
	}
	if resources.RequestMemoryMiB != 0 {
		result.RequestMemoryMiB = ptrInt(resources.RequestMemoryMiB)
		pc.record("agentsInitContainerResources.requestMemoryMiB")
	}
	if resources.LimitMemoryMiB != 0 {
		result.LimitMemoryMiB = ptrInt(resources.LimitMemoryMiB)
		pc.record("agentsInitContainerResources.limitMemoryMiB")
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

func convertCollectorGatewayToModel(gw *common.CollectorGatewayConfiguration, pc *provenanceCollector) (*model.CollectorGatewayConfig, error) {
	if gw == nil {
		return nil, nil
	}

	p := func(field string) string { return "collectorGateway." + field }
	result := &model.CollectorGatewayConfig{}

	setInt := func(val int, field string) *int {
		if val == 0 {
			return nil
		}
		pc.record(p(field))
		return ptrInt(val)
	}

	result.MinReplicas = setInt(gw.MinReplicas, "minReplicas")
	result.MaxReplicas = setInt(gw.MaxReplicas, "maxReplicas")
	result.RequestMemoryMiB = setInt(gw.RequestMemoryMiB, "requestMemoryMiB")
	result.LimitMemoryMiB = setInt(gw.LimitMemoryMiB, "limitMemoryMiB")
	result.RequestCPUm = setInt(gw.RequestCPUm, "requestCPUm")
	result.LimitCPUm = setInt(gw.LimitCPUm, "limitCPUm")
	result.MemoryLimiterLimitMiB = setInt(gw.MemoryLimiterLimitMiB, "memoryLimiterLimitMiB")
	result.MemoryLimiterSpikeLimitMiB = setInt(gw.MemoryLimiterSpikeLimitMiB, "memoryLimiterSpikeLimitMiB")
	result.GoMemLimitMiB = setInt(gw.GoMemLimitMib, "goMemLimitMiB")

	if gw.ClusterMetricsEnabled != nil {
		result.ClusterMetricsEnabled = gw.ClusterMetricsEnabled
		pc.record(p("clusterMetricsEnabled"))
	}
	if gw.HttpsProxyAddress != nil {
		result.HTTPSProxyAddress = gw.HttpsProxyAddress
		pc.record(p("httpsProxyAddress"))
	}

	if gw.ServiceGraph != nil && gw.ServiceGraph.Disabled != nil {
		result.ServiceGraphDisabled = gw.ServiceGraph.Disabled
		pc.record(p("serviceGraphDisabled"))
	}

	if gw.NodeSelector != nil && len(*gw.NodeSelector) > 0 {
		nodeSelectorJSON, err := json.Marshal(*gw.NodeSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal collectorGateway.nodeSelector: %w", err)
		}
		nodeSelectorStr := string(nodeSelectorJSON)
		result.NodeSelector = &nodeSelectorStr
		pc.record(p("nodeSelector"))
	}

	return result, nil
}

func convertCollectorNodeToModel(node *common.CollectorNodeConfiguration, pc *provenanceCollector) *model.CollectorNodeConfig {
	if node == nil {
		return nil
	}

	p := func(field string) string { return "collectorNode." + field }
	result := &model.CollectorNodeConfig{}

	setInt := func(val int, field string) *int {
		if val == 0 {
			return nil
		}
		pc.record(p(field))
		return ptrInt(val)
	}

	result.CollectorOwnMetricsPort = setInt(int(node.CollectorOwnMetricsPort), "collectorOwnMetricsPort")
	result.RequestMemoryMiB = setInt(node.RequestMemoryMiB, "requestMemoryMiB")
	result.LimitMemoryMiB = setInt(node.LimitMemoryMiB, "limitMemoryMiB")
	result.RequestCPUm = setInt(node.RequestCPUm, "requestCPUm")
	result.LimitCPUm = setInt(node.LimitCPUm, "limitCPUm")
	result.MemoryLimiterLimitMiB = setInt(node.MemoryLimiterLimitMiB, "memoryLimiterLimitMiB")
	result.MemoryLimiterSpikeLimitMiB = setInt(node.MemoryLimiterSpikeLimitMiB, "memoryLimiterSpikeLimitMiB")
	result.GoMemLimitMiB = setInt(node.GoMemLimitMib, "goMemLimitMiB")

	if node.EnableDataCompression != nil {
		result.EnableDataCompression = node.EnableDataCompression
		pc.record(p("enableDataCompression"))
	}

	if node.OtlpExporterConfiguration != nil {
		result.OtlpExporterConfiguration = convertOtlpExporterToModel(node.OtlpExporterConfiguration, pc)
	}

	return result
}

func convertOtlpExporterToModel(otlp *common.OtlpExporterConfiguration, pc *provenanceCollector) *model.OtlpExporterConfig {
	if otlp == nil {
		return nil
	}

	p := func(field string) string { return "collectorNode.otlpExporterConfiguration." + field }
	result := &model.OtlpExporterConfig{}

	if otlp.EnableDataCompression != nil {
		result.EnableDataCompression = otlp.EnableDataCompression
		pc.record(p("enableDataCompression"))
	}
	if otlp.Timeout != "" {
		result.Timeout = ptrStr(otlp.Timeout)
		pc.record(p("timeout"))
	}

	if otlp.RetryOnFailure != nil {
		rp := func(field string) string { return p("retryOnFailure." + field) }
		rf := &model.RetryOnFailureConfig{}
		if otlp.RetryOnFailure.Enabled != nil {
			rf.Enabled = otlp.RetryOnFailure.Enabled
			pc.record(rp("enabled"))
		}
		if otlp.RetryOnFailure.InitialInterval != "" {
			rf.InitialInterval = ptrStr(otlp.RetryOnFailure.InitialInterval)
			pc.record(rp("initialInterval"))
		}
		if otlp.RetryOnFailure.MaxInterval != "" {
			rf.MaxInterval = ptrStr(otlp.RetryOnFailure.MaxInterval)
			pc.record(rp("maxInterval"))
		}
		if otlp.RetryOnFailure.MaxElapsedTime != "" {
			rf.MaxElapsedTime = ptrStr(otlp.RetryOnFailure.MaxElapsedTime)
			pc.record(rp("maxElapsedTime"))
		}
		result.RetryOnFailure = rf
	}

	return result
}

func convertUserInstrumentationEnvsToModel(envs *common.UserInstrumentationEnvs, pc *provenanceCollector) (*model.UserInstrumentationEnvsConfig, error) {
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
		result.Languages = &languagesStr
		pc.record("userInstrumentationEnvs.languages")
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

func convertMetricsSourcesToModel(ms *common.MetricsSourceConfiguration, pc *provenanceCollector) *model.MetricsSourceConfig {
	if ms == nil {
		return nil
	}

	result := &model.MetricsSourceConfig{}

	if ms.SpanMetrics != nil {
		sm := ms.SpanMetrics
		p := func(f string) string { return "metricsSources.spanMetrics." + f }
		spanMetrics := &model.MetricsSourceSpanMetricsConfig{}

		if sm.Disabled != nil {
			spanMetrics.Disabled = sm.Disabled
			pc.record(p("disabled"))
		}
		if sm.Interval != "" {
			spanMetrics.Interval = ptrStr(sm.Interval)
			pc.record(p("interval"))
		}
		if sm.MetricsExpiration != "" {
			spanMetrics.MetricsExpiration = ptrStr(sm.MetricsExpiration)
			pc.record(p("metricsExpiration"))
		}
		spanMetrics.AdditionalDimensions = sm.AdditionalDimensions
		pc.record(p("additionalDimensions"))
		spanMetrics.HistogramBuckets = sm.ExplicitHistogramBuckets
		pc.record(p("histogramBuckets"))
		if sm.IncludedProcessInDimensions != nil {
			spanMetrics.IncludedProcessInDimensions = sm.IncludedProcessInDimensions
			pc.record(p("includedProcessInDimensions"))
		}
		spanMetrics.ExcludedResourceAttributes = sm.ExcludedResourceAttributes
		pc.record(p("excludedResourceAttributes"))
		spanMetrics.ResourceMetricsKeyAttributes = sm.ResourceMetricsKeyAttributes
		pc.record(p("resourceMetricsKeyAttributes"))
		if sm.HistogramDisabled {
			spanMetrics.HistogramDisabled = ptrBool(sm.HistogramDisabled)
			pc.record(p("histogramDisabled"))
		}

		result.SpanMetrics = spanMetrics
	}

	if ms.HostMetrics != nil {
		p := func(f string) string { return "metricsSources.hostMetrics." + f }
		hm := &model.MetricsSourceHostMetricsConfig{}
		if ms.HostMetrics.Disabled != nil {
			hm.Disabled = ms.HostMetrics.Disabled
			pc.record(p("disabled"))
		}
		if ms.HostMetrics.Interval != "" {
			hm.Interval = ptrStr(ms.HostMetrics.Interval)
			pc.record(p("interval"))
		}
		result.HostMetrics = hm
	}

	if ms.KubeletStats != nil {
		p := func(f string) string { return "metricsSources.kubeletStats." + f }
		ks := &model.MetricsSourceKubeletStatsConfig{}
		if ms.KubeletStats.Disabled != nil {
			ks.Disabled = ms.KubeletStats.Disabled
			pc.record(p("disabled"))
		}
		if ms.KubeletStats.Interval != "" {
			ks.Interval = ptrStr(ms.KubeletStats.Interval)
			pc.record(p("interval"))
		}
		result.KubeletStats = ks
	}

	if ms.OdigosOwnMetrics != nil {
		oom := &model.MetricsSourceOdigosOwnMetricsConfig{}
		if ms.OdigosOwnMetrics.Interval != "" {
			oom.Interval = ptrStr(ms.OdigosOwnMetrics.Interval)
			pc.record("metricsSources.odigosOwnMetrics.interval")
		}
		result.OdigosOwnMetrics = oom
	}

	if ms.AgentMetrics != nil {
		result.AgentMetrics = &model.MetricsSourceAgentMetricsConfig{}

		if ms.AgentMetrics.SpanMetrics != nil {
			result.AgentMetrics.SpanMetrics = &model.MetricsSourceAgentSpanMetricsConfig{
				Enabled: ptrBool(ms.AgentMetrics.SpanMetrics.Enabled),
			}
			pc.record("metricsSources.agentMetrics.spanMetrics.enabled")
		}

		if ms.AgentMetrics.RuntimeMetrics != nil && ms.AgentMetrics.RuntimeMetrics.Java != nil {
			javaConfig := &model.MetricsSourceAgentJavaRuntimeMetricsConfig{}
			if ms.AgentMetrics.RuntimeMetrics.Java.Disabled != nil {
				javaConfig.Disabled = ms.AgentMetrics.RuntimeMetrics.Java.Disabled
				pc.record("metricsSources.agentMetrics.runtimeMetrics.java.disabled")
			}

			if len(ms.AgentMetrics.RuntimeMetrics.Java.Metrics) > 0 {
				metrics := make([]*model.MetricsSourceAgentRuntimeMetricConfig, len(ms.AgentMetrics.RuntimeMetrics.Java.Metrics))
				for i, m := range ms.AgentMetrics.RuntimeMetrics.Java.Metrics {
					basePath := fmt.Sprintf("metricsSources.agentMetrics.runtimeMetrics.java.metrics.%d", i)
					metrics[i] = &model.MetricsSourceAgentRuntimeMetricConfig{
						Name: ptrStr(m.Name),
					}
					pc.record(basePath + ".name")
					if m.Disabled != nil {
						metrics[i].Disabled = m.Disabled
						pc.record(basePath + ".disabled")
					}
				}
				javaConfig.Metrics = metrics
			}

			result.AgentMetrics.RuntimeMetrics = &model.MetricsSourceAgentRuntimeMetricsConfig{
				Java: javaConfig,
			}
		}
	}

	return result
}
