package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func UpdateLocalUIConfig(ctx context.Context, c client.Client, input model.LocalUIConfigInput) error {
	ns := env.GetCurrentNamespace()

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var cm v1.ConfigMap
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosLocalUiConfigName}, &cm)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return createLocalUiConfigMap(ctx, c, ns, input)
			}
			return err
		}

		cfg := common.OdigosConfiguration{}
		if cm.Data != nil && cm.Data[consts.OdigosConfigurationFileName] != "" {
			if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg); err != nil {
				return fmt.Errorf("parse existing config: %w", err)
			}
		}

		applyLocalUiConfigInput(&cfg, input)

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[consts.OdigosConfigurationFileName] = string(data)
		return c.Update(ctx, &cm)
	})
}

func createLocalUiConfigMap(ctx context.Context, c client.Client, ns string, input model.LocalUIConfigInput) error {
	ownerCm := v1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosConfigurationName}, &ownerCm); err != nil {
		return fmt.Errorf("failed to get odigos-configuration for owner reference: %w", err)
	}

	cfg := common.OdigosConfiguration{}
	applyLocalUiConfigInput(&cfg, input)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	newCm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosLocalUiConfigName,
			Namespace: ns,
			Labels:    map[string]string{k8sconsts.OdigosSystemConfigLabelKey: "local-ui"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1", Kind: "ConfigMap", Name: ownerCm.Name, UID: ownerCm.UID,
			}},
		},
		Data: map[string]string{consts.OdigosConfigurationFileName: string(data)},
	}
	return c.Create(ctx, &newCm)
}

func ResetLocalUiConfigToFactoryDefaults(ctx context.Context, c client.Client) error {
	ns := env.GetCurrentNamespace()

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var cm v1.ConfigMap
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosLocalUiConfigName}, &cm)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		emptyCfg := common.OdigosConfiguration{}
		data, err := yaml.Marshal(emptyCfg)
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[consts.OdigosConfigurationFileName] = string(data)
		return c.Update(ctx, &cm)
	})
}

func applyLocalUiConfigInput(cfg *common.OdigosConfiguration, input model.LocalUIConfigInput) {
	if input.TelemetryEnabled != nil {
		cfg.TelemetryEnabled = *input.TelemetryEnabled
	}
	if input.IgnoredNamespaces != nil {
		cfg.IgnoredNamespaces = input.IgnoredNamespaces
	}
	if input.IgnoredContainers != nil {
		cfg.IgnoredContainers = input.IgnoredContainers
	}
	if input.IgnoreOdigosNamespace != nil {
		cfg.IgnoreOdigosNamespace = input.IgnoreOdigosNamespace
	}
	if input.ClusterName != nil {
		cfg.ClusterName = *input.ClusterName
	}
	if input.Instrumentor != nil {
		if m := convertEnvInjectionMethodToCommon(input.Instrumentor.AgentEnvVarsInjectionMethod); m != nil {
			cfg.AgentEnvVarsInjectionMethod = m
		}
		if input.Instrumentor.CheckDeviceHealthBeforeInjection != nil {
			cfg.CheckDeviceHealthBeforeInjection = input.Instrumentor.CheckDeviceHealthBeforeInjection
		}
	}
	if input.AllowConcurrentAgents != nil {
		if input.AllowConcurrentAgents.Enabled != nil {
			cfg.AllowConcurrentAgents = input.AllowConcurrentAgents.Enabled
		}
	}
	if input.Wasp != nil {
		if input.Wasp.Enabled != nil {
			cfg.WaspEnabled = input.Wasp.Enabled
		}
	}
	if input.Rollout != nil {
		if cfg.Rollout == nil {
			cfg.Rollout = &common.RolloutConfiguration{}
		}
		if input.Rollout.AutomaticRolloutDisabled != nil {
			cfg.Rollout.AutomaticRolloutDisabled = input.Rollout.AutomaticRolloutDisabled
		}
		if input.Rollout.MaxConcurrentRollouts != nil {
			cfg.Rollout.MaxConcurrentRollouts = *input.Rollout.MaxConcurrentRollouts
		}
	}
	if input.AutoRollback != nil {
		if input.AutoRollback.Disabled != nil {
			cfg.RollbackDisabled = input.AutoRollback.Disabled
		}
		if input.AutoRollback.GraceTime != nil {
			cfg.RollbackGraceTime = *input.AutoRollback.GraceTime
		}
		if input.AutoRollback.StabilityWindowTime != nil {
			cfg.RollbackStabilityWindow = *input.AutoRollback.StabilityWindowTime
		}
	}
	if input.GoAutoOffsetsCron != nil {
		cfg.GoAutoOffsetsCron = *input.GoAutoOffsetsCron
	}
	if input.GoAutoOffsetsMode != nil {
		cfg.GoAutoOffsetsMode = *input.GoAutoOffsetsMode
	}
	if input.Sampling != nil {
		if cfg.Sampling == nil {
			cfg.Sampling = &common.SamplingConfiguration{}
		}
		applySamplingInput(cfg.Sampling, input.Sampling)
	}
	if input.ComponentLogLevels != nil {
		if cfg.ComponentLogLevels == nil {
			cfg.ComponentLogLevels = &common.ComponentLogLevels{}
		}
		applyComponentLogLevelsInput(cfg.ComponentLogLevels, input.ComponentLogLevels)
	}
}

func applySamplingInput(cfg *common.SamplingConfiguration, input *model.LocalUIConfigSamplingInput) {
	if input.DryRun != nil {
		cfg.DryRun = input.DryRun
	}
	if input.SpanSamplingAttributes != nil {
		if cfg.SpanSamplingAttributes == nil {
			cfg.SpanSamplingAttributes = &sampling.SpanSamplingAttributesConfiguration{}
		}
		if input.SpanSamplingAttributes.Disabled != nil {
			cfg.SpanSamplingAttributes.Disabled = input.SpanSamplingAttributes.Disabled
		}
		if input.SpanSamplingAttributes.SamplingCategoryDisabled != nil {
			cfg.SpanSamplingAttributes.SamplingCategoryDisabled = input.SpanSamplingAttributes.SamplingCategoryDisabled
		}
		if input.SpanSamplingAttributes.TraceDecidingRuleDisabled != nil {
			cfg.SpanSamplingAttributes.TraceDecidingRuleDisabled = input.SpanSamplingAttributes.TraceDecidingRuleDisabled
		}
		if input.SpanSamplingAttributes.SpanDecisionAttributesDisabled != nil {
			cfg.SpanSamplingAttributes.SpanDecisionAttributesDisabled = input.SpanSamplingAttributes.SpanDecisionAttributesDisabled
		}
	}
	if input.TailSampling != nil {
		if cfg.TailSampling == nil {
			cfg.TailSampling = &sampling.TailSamplingConfiguration{}
		}
		if input.TailSampling.Disabled != nil {
			cfg.TailSampling.Disabled = input.TailSampling.Disabled
		}
		if input.TailSampling.TraceAggregationWaitDuration != nil {
			cfg.TailSampling.TraceAggregationWaitDuration = input.TailSampling.TraceAggregationWaitDuration
		}
	}
	if input.K8sHealthProbesSampling != nil {
		if cfg.K8sHealthProbesSampling == nil {
			cfg.K8sHealthProbesSampling = &common.K8sHealthProbesSamplingConfiguration{}
		}
		if input.K8sHealthProbesSampling.Enabled != nil {
			cfg.K8sHealthProbesSampling.Enabled = input.K8sHealthProbesSampling.Enabled
		}
		if input.K8sHealthProbesSampling.KeepPercentage != nil {
			cfg.K8sHealthProbesSampling.KeepPercentage = input.K8sHealthProbesSampling.KeepPercentage
		}
	}
}

func applyComponentLogLevelsInput(cfg *common.ComponentLogLevels, input *model.LocalUIConfigComponentLogLevelsInput) {
	if input.Default != nil {
		cfg.Default = common.OdigosLogLevel(*input.Default)
	}
	if input.Autoscaler != nil {
		cfg.Autoscaler = common.OdigosLogLevel(*input.Autoscaler)
	}
	if input.Scheduler != nil {
		cfg.Scheduler = common.OdigosLogLevel(*input.Scheduler)
	}
	if input.Instrumentor != nil {
		cfg.Instrumentor = common.OdigosLogLevel(*input.Instrumentor)
	}
	if input.Odiglet != nil {
		cfg.Odiglet = common.OdigosLogLevel(*input.Odiglet)
	}
	if input.Deviceplugin != nil {
		cfg.Deviceplugin = common.OdigosLogLevel(*input.Deviceplugin)
	}
	if input.UI != nil {
		cfg.UI = common.OdigosLogLevel(*input.UI)
	}
	if input.Collector != nil {
		cfg.Collector = common.OdigosLogLevel(*input.Collector)
	}
}

// convertEnvInjectionMethodToCommon maps the GraphQL EnvInjectionMethod enum
// (underscores, e.g. "pod_manifest") to the common.EnvInjectionMethod value
// (hyphens, e.g. "pod-manifest") expected by the ConfigMap / Helm.
func convertEnvInjectionMethodToCommon(m *model.EnvInjectionMethod) *common.EnvInjectionMethod {
	if m == nil {
		return nil
	}
	var result common.EnvInjectionMethod
	switch *m {
	case model.EnvInjectionMethodPodManifest:
		result = common.PodManifestEnvInjectionMethod
	case model.EnvInjectionMethodLoaderFallbackToPodManifest:
		result = common.LoaderFallbackToPodManifestInjectionMethod
	default:
		result = common.LoaderEnvInjectionMethod
	}
	return &result
}
