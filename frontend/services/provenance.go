package services

import (
	"context"
	"reflect"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComputeProvenance calculates which ConfigMap each field in the effective config
// originated from, by reading the individual source ConfigMaps and comparing.
// This is computed on-demand rather than stored alongside the effective config.
func ComputeProvenance(ctx context.Context, c client.Client, effectiveConfig *common.OdigosConfiguration) (map[string]string, error) {
	if effectiveConfig == nil {
		return nil, nil
	}

	provenance := make(map[string]string)

	remoteConfig, _ := GetRemoteConfig(ctx, c)
	localUIConfig, _ := GetLocalUIConfig(ctx, c)
	baseConfig, _ := GetHelmDeploymentConfig(ctx, c)

	// Record overlay provenance in precedence order: remote first, then local UI overwrites.
	recordOverlayProvenance(remoteConfig, provenance, consts.OdigosRemoteConfigName)
	recordOverlayProvenance(localUIConfig, provenance, consts.OdigosLocalUiConfigName)

	detectProfileProvenance(baseConfig, remoteConfig, localUIConfig, effectiveConfig, provenance)

	return provenance, nil
}

// recordOverlayProvenance checks which fields are set in an overlay config
// and records provenance for them. This mirrors the field checks in the
// scheduler controller's mergeConfigs function.
func recordOverlayProvenance(config *common.OdigosConfiguration, provenance map[string]string, sourceName string) {
	if config == nil {
		return
	}

	if config.TelemetryEnabled {
		provenance["telemetryEnabled"] = sourceName
	}
	if config.IgnoredNamespaces != nil {
		provenance["ignoredNamespaces"] = sourceName
	}
	if config.IgnoredContainers != nil {
		provenance["ignoredContainers"] = sourceName
	}
	if config.IgnoreOdigosNamespace != nil {
		provenance["ignoreOdigosNamespace"] = sourceName
	}
	if config.ClusterName != "" {
		provenance["clusterName"] = sourceName
	}
	if config.AgentEnvVarsInjectionMethod != nil {
		provenance["agentEnvVarsInjectionMethod"] = sourceName
	}
	if config.CheckDeviceHealthBeforeInjection != nil {
		provenance["checkDeviceHealthBeforeInjection"] = sourceName
	}
	if config.AllowConcurrentAgents != nil {
		provenance["allowConcurrentAgents"] = sourceName
	}
	if config.WaspEnabled != nil {
		provenance["waspEnabled"] = sourceName
	}
	if config.Rollout != nil {
		if config.Rollout.AutomaticRolloutDisabled != nil {
			provenance["rollout.automaticRolloutDisabled"] = sourceName
		}
		if config.Rollout.MaxConcurrentRollouts != 0 {
			provenance["rollout.maxConcurrentRollouts"] = sourceName
		}
	}
	if config.RollbackDisabled != nil {
		provenance["rollbackDisabled"] = sourceName
	}
	if config.RollbackGraceTime != "" {
		provenance["rollbackGraceTime"] = sourceName
	}
	if config.RollbackStabilityWindow != "" {
		provenance["rollbackStabilityWindow"] = sourceName
	}
	if config.GoAutoOffsetsCron != "" {
		provenance["goAutoOffsetsCron"] = sourceName
	}
	if config.GoAutoOffsetsMode != "" {
		provenance["goAutoOffsetsMode"] = sourceName
	}

	if config.Sampling != nil {
		if config.Sampling.DryRun != nil {
			provenance["sampling.dryRun"] = sourceName
		}
		if config.Sampling.SpanSamplingAttributes != nil {
			if config.Sampling.SpanSamplingAttributes.Disabled != nil {
				provenance["sampling.spanSamplingAttributes.disabled"] = sourceName
			}
			if config.Sampling.SpanSamplingAttributes.SamplingCategoryDisabled != nil {
				provenance["sampling.spanSamplingAttributes.samplingCategoryDisabled"] = sourceName
			}
			if config.Sampling.SpanSamplingAttributes.TraceDecidingRuleDisabled != nil {
				provenance["sampling.spanSamplingAttributes.traceDecidingRuleDisabled"] = sourceName
			}
			if config.Sampling.SpanSamplingAttributes.SpanDecisionAttributesDisabled != nil {
				provenance["sampling.spanSamplingAttributes.spanDecisionAttributesDisabled"] = sourceName
			}
		}
		if config.Sampling.TailSampling != nil {
			if config.Sampling.TailSampling.Disabled != nil {
				provenance["sampling.tailSampling.disabled"] = sourceName
			}
			if config.Sampling.TailSampling.TraceAggregationWaitDuration != nil {
				provenance["sampling.tailSampling.traceAggregationWaitDuration"] = sourceName
			}
		}
		if config.Sampling.K8sHealthProbesSampling != nil {
			if config.Sampling.K8sHealthProbesSampling.Enabled != nil {
				provenance["sampling.k8sHealthProbesSampling.enabled"] = sourceName
			}
			if config.Sampling.K8sHealthProbesSampling.KeepPercentage != nil {
				provenance["sampling.k8sHealthProbesSampling.keepPercentage"] = sourceName
			}
		}
	}

	if config.ComponentLogLevels != nil {
		src := config.ComponentLogLevels
		if src.Default != "" {
			provenance["componentLogLevels.default"] = sourceName
		}
		if src.Autoscaler != "" {
			provenance["componentLogLevels.autoscaler"] = sourceName
		}
		if src.Scheduler != "" {
			provenance["componentLogLevels.scheduler"] = sourceName
		}
		if src.Instrumentor != "" {
			provenance["componentLogLevels.instrumentor"] = sourceName
		}
		if src.Odiglet != "" {
			provenance["componentLogLevels.odiglet"] = sourceName
		}
		if src.Deviceplugin != "" {
			provenance["componentLogLevels.deviceplugin"] = sourceName
		}
		if src.UI != "" {
			provenance["componentLogLevels.ui"] = sourceName
		}
		if src.Collector != "" {
			provenance["componentLogLevels.collector"] = sourceName
		}
	}
}

// detectProfileProvenance identifies fields modified by profile application.
// For each profile-modifiable field, it computes what the pre-profile merged value
// would be (base + overlays) and compares against the effective config.
func detectProfileProvenance(base, remote, local, effective *common.OdigosConfiguration, provenance map[string]string) {
	if base == nil || effective == nil {
		return
	}

	// rollbackDisabled: settable by overlays
	preRollbackDisabled := base.RollbackDisabled
	if remote != nil && remote.RollbackDisabled != nil {
		preRollbackDisabled = remote.RollbackDisabled
	}
	if local != nil && local.RollbackDisabled != nil {
		preRollbackDisabled = local.RollbackDisabled
	}
	if !reflect.DeepEqual(preRollbackDisabled, effective.RollbackDisabled) {
		provenance["rollbackDisabled"] = "profile"
	}

	// mountMethod: only base config and profiles can set this.
	// Compare resolved values since the controller applies a default for nil.
	if resolvedMountMethod(base.MountMethod) != resolvedMountMethod(effective.MountMethod) {
		provenance["mountMethod"] = "profile"
	}

	// agentEnvVarsInjectionMethod: settable by overlays.
	// Compare resolved values since the controller applies a default for nil.
	preEnvInj := base.AgentEnvVarsInjectionMethod
	if remote != nil && remote.AgentEnvVarsInjectionMethod != nil {
		preEnvInj = remote.AgentEnvVarsInjectionMethod
	}
	if local != nil && local.AgentEnvVarsInjectionMethod != nil {
		preEnvInj = local.AgentEnvVarsInjectionMethod
	}
	if resolvedEnvInjectionMethod(preEnvInj) != resolvedEnvInjectionMethod(effective.AgentEnvVarsInjectionMethod) {
		provenance["agentEnvVarsInjectionMethod"] = "profile"
	}

	// allowConcurrentAgents: settable by overlays
	preAllowConcurrent := base.AllowConcurrentAgents
	if remote != nil && remote.AllowConcurrentAgents != nil {
		preAllowConcurrent = remote.AllowConcurrentAgents
	}
	if local != nil && local.AllowConcurrentAgents != nil {
		preAllowConcurrent = local.AllowConcurrentAgents
	}
	if !reflect.DeepEqual(preAllowConcurrent, effective.AllowConcurrentAgents) {
		provenance["allowConcurrentAgents"] = "profile"
	}

	// checkDeviceHealthBeforeInjection: settable by overlays
	preCheckHealth := base.CheckDeviceHealthBeforeInjection
	if remote != nil && remote.CheckDeviceHealthBeforeInjection != nil {
		preCheckHealth = remote.CheckDeviceHealthBeforeInjection
	}
	if local != nil && local.CheckDeviceHealthBeforeInjection != nil {
		preCheckHealth = local.CheckDeviceHealthBeforeInjection
	}
	if !reflect.DeepEqual(preCheckHealth, effective.CheckDeviceHealthBeforeInjection) {
		provenance["checkDeviceHealthBeforeInjection"] = "profile"
	}

	// waspEnabled: settable by overlays
	preWasp := base.WaspEnabled
	if remote != nil && remote.WaspEnabled != nil {
		preWasp = remote.WaspEnabled
	}
	if local != nil && local.WaspEnabled != nil {
		preWasp = local.WaspEnabled
	}
	if !reflect.DeepEqual(preWasp, effective.WaspEnabled) {
		provenance["waspEnabled"] = "profile"
	}

	// metricsSources: only base config and profiles can set this
	if !reflect.DeepEqual(base.MetricsSources, effective.MetricsSources) {
		provenance["metricsSources"] = "profile"
	}
}

func resolvedMountMethod(m *common.MountMethod) common.MountMethod {
	if m == nil {
		return common.K8sVirtualDeviceMountMethod
	}
	return *m
}

func resolvedEnvInjectionMethod(m *common.EnvInjectionMethod) common.EnvInjectionMethod {
	if m == nil {
		return common.LoaderFallbackToPodManifestInjectionMethod
	}
	return *m
}
