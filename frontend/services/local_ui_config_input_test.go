package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"
)

// applyLocalUiConfigInput is the write half of the settings screen: it folds a partial
// GraphQL input into the odigos-local-ui-config document that the scheduler merges over
// the helm baseline. Every field is guarded by its own nil check, so a missed guard either
// discards the user's edit or clobbers an unrelated setting with a zero value - both are
// silent, and both survive until someone notices the setting "does not stick".

const localUiConfigTestNamespace = "odigos-local-ui-test"

func localUiBoolPtr(v bool) *bool        { return &v }
func localUiIntPtr(v int) *int           { return &v }
func localUiStrPtr(v string) *string     { return &v }
func localUiFloatPtr(v float64) *float64 { return &v }

func localUiLogLevelPtr(v model.OdigosLogLevel) *model.OdigosLogLevel { return &v }

// fullLocalUiConfigInput sets every field the GraphQL input exposes, with values that are
// distinguishable from both the zero value and from each other.
func fullLocalUiConfigInput() model.LocalUIConfigInput {
	envInjection := model.EnvInjectionMethodPodManifest
	return model.LocalUIConfigInput{
		TelemetryEnabled:      localUiBoolPtr(true),
		IgnoredNamespaces:     []string{"payments", "kube-system"},
		IgnoredContainers:     []string{"istio-proxy"},
		IgnoreOdigosNamespace: localUiBoolPtr(true),
		ClusterName:           localUiStrPtr("prod-eu-west"),
		Instrumentor: &model.LocalUIConfigInstrumentorInput{
			AgentEnvVarsInjectionMethod:      &envInjection,
			CheckDeviceHealthBeforeInjection: localUiBoolPtr(true),
		},
		AllowConcurrentAgents: &model.LocalUIConfigAllowConcurrentAgentsInput{
			Enabled: localUiBoolPtr(true),
		},
		Rollout: &model.LocalUIConfigRolloutInput{
			AutomaticRolloutDisabled: localUiBoolPtr(true),
			MaxConcurrentRollouts:    localUiIntPtr(7),
		},
		AutoRollback: &model.LocalUIConfigAutoRollbackInput{
			Disabled:            localUiBoolPtr(true),
			GraceTime:           localUiStrPtr("3m"),
			StabilityWindowTime: localUiStrPtr("9m"),
		},
		GoAutoOffsetsCron: localUiStrPtr("0 3 * * *"),
		GoAutoOffsetsMode: localUiStrPtr("cron"),
		Sampling: &model.LocalUIConfigSamplingInput{
			DryRun: localUiBoolPtr(true),
			SpanSamplingAttributes: &model.LocalUIConfigSpanSamplingAttributesInput{
				Disabled:                       localUiBoolPtr(true),
				SamplingCategoryDisabled:       localUiBoolPtr(true),
				TraceDecidingRuleDisabled:      localUiBoolPtr(true),
				SpanDecisionAttributesDisabled: localUiBoolPtr(true),
			},
			TailSampling: &model.TailSamplingConfigInput{
				Disabled:                     localUiBoolPtr(true),
				TraceAggregationWaitDuration: localUiStrPtr("11s"),
			},
			K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{
				Enabled:        localUiBoolPtr(true),
				KeepPercentage: localUiFloatPtr(12.5),
			},
		},
		ComponentLogLevels: &model.LocalUIConfigComponentLogLevelsInput{
			Default:      localUiLogLevelPtr(model.OdigosLogLevelWarn),
			Autoscaler:   localUiLogLevelPtr(model.OdigosLogLevelDebug),
			Scheduler:    localUiLogLevelPtr(model.OdigosLogLevelError),
			Instrumentor: localUiLogLevelPtr(model.OdigosLogLevelDebug),
			Odiglet:      localUiLogLevelPtr(model.OdigosLogLevelWarn),
			Deviceplugin: localUiLogLevelPtr(model.OdigosLogLevelError),
			UI:           localUiLogLevelPtr(model.OdigosLogLevelDebug),
			Collector:    localUiLogLevelPtr(model.OdigosLogLevelWarn),
		},
		TraceCorrelations: &model.LocalUIConfigTraceCorrelationsInput{
			ServiceIo: &model.LocalUIConfigTraceCorrelationsServiceIOInput{
				Enabled:              localUiBoolPtr(true),
				InputSpanAttributes:  []string{"http.route"},
				OutputSpanAttributes: []string{"db.system"},
				MetricsFlushInterval: localUiStrPtr("45s"),
			},
		},
	}
}

func TestApplyLocalUiConfigInput_WritesEveryInputField(t *testing.T) {
	podManifest := common.PodManifestEnvInjectionMethod
	cfg := &common.OdigosConfiguration{}

	applyLocalUiConfigInput(cfg, fullLocalUiConfigInput())

	assert.Equal(t, &common.OdigosConfiguration{
		TelemetryEnabled:                 true,
		IgnoredNamespaces:                []string{"payments", "kube-system"},
		IgnoredContainers:                []string{"istio-proxy"},
		IgnoreOdigosNamespace:            localUiBoolPtr(true),
		ClusterName:                      "prod-eu-west",
		AgentEnvVarsInjectionMethod:      &podManifest,
		CheckDeviceHealthBeforeInjection: localUiBoolPtr(true),
		AllowConcurrentAgents:            localUiBoolPtr(true),
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: localUiBoolPtr(true),
			MaxConcurrentRollouts:    7,
		},
		RollbackDisabled:        localUiBoolPtr(true),
		RollbackGraceTime:       "3m",
		RollbackStabilityWindow: "9m",
		GoAutoOffsetsCron:       "0 3 * * *",
		GoAutoOffsetsMode:       "cron",
		Sampling: &common.SamplingConfiguration{
			DryRun: localUiBoolPtr(true),
			SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{
				Disabled:                       localUiBoolPtr(true),
				SamplingCategoryDisabled:       localUiBoolPtr(true),
				TraceDecidingRuleDisabled:      localUiBoolPtr(true),
				SpanDecisionAttributesDisabled: localUiBoolPtr(true),
			},
			TailSampling: &commonapisampling.TailSamplingConfiguration{
				Disabled:                     localUiBoolPtr(true),
				TraceAggregationWaitDuration: localUiStrPtr("11s"),
			},
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        localUiBoolPtr(true),
				KeepPercentage: localUiFloatPtr(12.5),
			},
		},
		ComponentLogLevels: &common.ComponentLogLevels{
			Default:      common.LogLevelWarn,
			Autoscaler:   common.LogLevelDebug,
			Scheduler:    common.LogLevelError,
			Instrumentor: common.LogLevelDebug,
			Odiglet:      common.LogLevelWarn,
			Deviceplugin: common.LogLevelError,
			UI:           common.LogLevelDebug,
			Collector:    common.LogLevelWarn,
		},
		TraceCorrelations: &common.TraceCorrelationsConfiguration{
			ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
				Enabled:              localUiBoolPtr(true),
				InputSpanAttributes:  []string{"http.route"},
				OutputSpanAttributes: []string{"db.system"},
				MetricsFlushInterval: "45s",
			},
		},
	}, cfg)
}

// The UI sends only the fields the user touched. An empty input must leave the stored
// document byte-for-byte identical, otherwise saving one setting resets the others.
func TestApplyLocalUiConfigInput_EmptyInputTouchesNothing(t *testing.T) {
	cfg := &common.OdigosConfiguration{}
	applyLocalUiConfigInput(cfg, model.LocalUIConfigInput{})
	assert.Equal(t, &common.OdigosConfiguration{}, cfg)

	existing := fullyPopulatedOverlayConfig()
	applyLocalUiConfigInput(existing, model.LocalUIConfigInput{})
	assert.Equal(t, fullyPopulatedOverlayConfig(), existing)
}

// The nested inputs are containers: an input that names a block but sets no field inside
// it must not zero out what is already stored there.
func TestApplyLocalUiConfigInput_EmptyNestedInputsPreserveStoredValues(t *testing.T) {
	before := fullyPopulatedOverlayConfig()
	cfg := fullyPopulatedOverlayConfig()

	applyLocalUiConfigInput(cfg, model.LocalUIConfigInput{
		Instrumentor:          &model.LocalUIConfigInstrumentorInput{},
		AllowConcurrentAgents: &model.LocalUIConfigAllowConcurrentAgentsInput{},
		Rollout:               &model.LocalUIConfigRolloutInput{},
		AutoRollback:          &model.LocalUIConfigAutoRollbackInput{},
		Sampling: &model.LocalUIConfigSamplingInput{
			SpanSamplingAttributes:  &model.LocalUIConfigSpanSamplingAttributesInput{},
			TailSampling:            &model.TailSamplingConfigInput{},
			K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{},
		},
		ComponentLogLevels: &model.LocalUIConfigComponentLogLevelsInput{},
		TraceCorrelations: &model.LocalUIConfigTraceCorrelationsInput{
			ServiceIo: &model.LocalUIConfigTraceCorrelationsServiceIOInput{},
		},
	})

	assert.Equal(t, before, cfg)
}

// A nested block named in the input but absent from the stored document is created, so
// the first edit of a sampling or log-level setting is not dropped.
func TestApplyLocalUiConfigInput_CreatesMissingNestedBlocks(t *testing.T) {
	cfg := &common.OdigosConfiguration{}

	applyLocalUiConfigInput(cfg, model.LocalUIConfigInput{
		Rollout:            &model.LocalUIConfigRolloutInput{},
		Sampling:           &model.LocalUIConfigSamplingInput{},
		ComponentLogLevels: &model.LocalUIConfigComponentLogLevelsInput{},
	})

	assert.Equal(t, &common.OdigosConfiguration{
		Rollout:            &common.RolloutConfiguration{},
		Sampling:           &common.SamplingConfiguration{},
		ComponentLogLevels: &common.ComponentLogLevels{},
	}, cfg)
}

// Editing one setting must not disturb its neighbours in the same block.
func TestApplyLocalUiConfigInput_PartialEditKeepsSiblingsInTheSameBlock(t *testing.T) {
	cfg := &common.OdigosConfiguration{
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: localUiBoolPtr(true),
			MaxConcurrentRollouts:    7,
		},
		RollbackGraceTime:       "3m",
		RollbackStabilityWindow: "9m",
	}

	applyLocalUiConfigInput(cfg, model.LocalUIConfigInput{
		Rollout:      &model.LocalUIConfigRolloutInput{MaxConcurrentRollouts: localUiIntPtr(2)},
		AutoRollback: &model.LocalUIConfigAutoRollbackInput{GraceTime: localUiStrPtr("30s")},
	})

	assert.Equal(t, localUiBoolPtr(true), cfg.Rollout.AutomaticRolloutDisabled)
	assert.Equal(t, 2, cfg.Rollout.MaxConcurrentRollouts)
	assert.Equal(t, "30s", cfg.RollbackGraceTime)
	assert.Equal(t, "9m", cfg.RollbackStabilityWindow)
}

// The instrumentor block routes its env injection method through an enum bridge that
// returns nil for an unset value; that nil must not overwrite a stored method.
func TestApplyLocalUiConfigInput_UnsetEnvInjectionMethodKeepsTheStoredOne(t *testing.T) {
	loader := common.LoaderEnvInjectionMethod
	cfg := &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &loader}

	applyLocalUiConfigInput(cfg, model.LocalUIConfigInput{
		Instrumentor: &model.LocalUIConfigInstrumentorInput{
			CheckDeviceHealthBeforeInjection: localUiBoolPtr(true),
		},
	})

	require.NotNil(t, cfg.AgentEnvVarsInjectionMethod)
	assert.Equal(t, common.LoaderEnvInjectionMethod, *cfg.AgentEnvVarsInjectionMethod)
}

func TestConvertEnvInjectionMethodToCommon(t *testing.T) {
	tests := []struct {
		name  string
		input *model.EnvInjectionMethod
		want  *common.EnvInjectionMethod
	}{
		{"nil stays nil", nil, nil},
		{
			"pod_manifest",
			envInjectionModel(model.EnvInjectionMethodPodManifest),
			envInjectionCommon(common.PodManifestEnvInjectionMethod),
		},
		{
			"loader_fallback_to_pod_manifest",
			envInjectionModel(model.EnvInjectionMethodLoaderFallbackToPodManifest),
			envInjectionCommon(common.LoaderFallbackToPodManifestInjectionMethod),
		},
		{
			"loader",
			envInjectionModel(model.EnvInjectionMethodLoader),
			envInjectionCommon(common.LoaderEnvInjectionMethod),
		},
		{
			"an unrecognised enum value falls back to loader",
			envInjectionModel(model.EnvInjectionMethod("not-a-method")),
			envInjectionCommon(common.LoaderEnvInjectionMethod),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, convertEnvInjectionMethodToCommon(tt.input))
		})
	}
}

func envInjectionModel(m model.EnvInjectionMethod) *model.EnvInjectionMethod    { return &m }
func envInjectionCommon(m common.EnvInjectionMethod) *common.EnvInjectionMethod { return &m }

// The bridge falls through to `loader` for anything it does not recognise, so a GraphQL
// enum value added without a matching case is accepted and then quietly injects agent env
// vars the wrong way. Two enum values collapsing onto one common value is the observable
// symptom, and the underscore-to-hyphen spelling is the whole point of the bridge.
func TestConvertEnvInjectionMethodToCommon_EveryEnumValueHasItsOwnCase(t *testing.T) {
	seen := map[common.EnvInjectionMethod]model.EnvInjectionMethod{}

	for _, m := range model.AllEnvInjectionMethod {
		converted := convertEnvInjectionMethodToCommon(&m)
		require.NotNil(t, converted)

		assert.Equal(t, strings.ReplaceAll(string(m), "_", "-"), string(*converted))

		previous, collided := seen[*converted]
		assert.False(t, collided,
			"%q and %q both convert to %q, so one of them is falling through the default case",
			previous, m, *converted)
		seen[*converted] = m
	}
}

// Log levels are passed straight through as strings, so a GraphQL level that the common
// package does not know about would be written into the ConfigMap and rejected by every
// component that reads it.
func TestApplyComponentLogLevelsInput_EveryGraphQLLogLevelIsAKnownCommonLevel(t *testing.T) {
	knownLevels := map[common.OdigosLogLevel]struct{}{
		common.LogLevelError: {},
		common.LogLevelWarn:  {},
		common.LogLevelInfo:  {},
		common.LogLevelDebug: {},
	}

	for _, level := range model.AllOdigosLogLevel {
		t.Run(string(level), func(t *testing.T) {
			cfg := &common.ComponentLogLevels{}

			applyComponentLogLevelsInput(cfg, &model.LocalUIConfigComponentLogLevelsInput{Default: &level})

			assert.Contains(t, knownLevels, cfg.Default)
			assert.Equal(t, string(level), string(cfg.Default))
		})
	}
}

func TestApplyComponentLogLevelsInput_WritesOnlyTheProvidedComponents(t *testing.T) {
	cfg := &common.ComponentLogLevels{
		Default:    common.LogLevelInfo,
		Autoscaler: common.LogLevelWarn,
	}

	applyComponentLogLevelsInput(cfg, &model.LocalUIConfigComponentLogLevelsInput{
		Odiglet: localUiLogLevelPtr(model.OdigosLogLevelDebug),
	})

	assert.Equal(t, &common.ComponentLogLevels{
		Default:    common.LogLevelInfo,
		Autoscaler: common.LogLevelWarn,
		Odiglet:    common.LogLevelDebug,
	}, cfg)
}

// Each component reads its own input field. A copy-paste that wires two components to the
// same input leaves the map the same size and only shows up as "the wrong service got the
// debug level".
func TestApplyComponentLogLevelsInput_EachComponentReadsItsOwnField(t *testing.T) {
	tests := []struct {
		name  string
		input *model.LocalUIConfigComponentLogLevelsInput
		want  *common.ComponentLogLevels
	}{
		{"default", &model.LocalUIConfigComponentLogLevelsInput{Default: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{Default: common.LogLevelDebug}},
		{"autoscaler", &model.LocalUIConfigComponentLogLevelsInput{Autoscaler: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{Autoscaler: common.LogLevelDebug}},
		{"scheduler", &model.LocalUIConfigComponentLogLevelsInput{Scheduler: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{Scheduler: common.LogLevelDebug}},
		{"instrumentor", &model.LocalUIConfigComponentLogLevelsInput{Instrumentor: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{Instrumentor: common.LogLevelDebug}},
		{"odiglet", &model.LocalUIConfigComponentLogLevelsInput{Odiglet: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{Odiglet: common.LogLevelDebug}},
		{"deviceplugin", &model.LocalUIConfigComponentLogLevelsInput{Deviceplugin: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{Deviceplugin: common.LogLevelDebug}},
		{"ui", &model.LocalUIConfigComponentLogLevelsInput{UI: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{UI: common.LogLevelDebug}},
		{"collector", &model.LocalUIConfigComponentLogLevelsInput{Collector: localUiLogLevelPtr(model.OdigosLogLevelDebug)}, &common.ComponentLogLevels{Collector: common.LogLevelDebug}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &common.ComponentLogLevels{}

			applyComponentLogLevelsInput(cfg, tt.input)

			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestApplySamplingInput_EachFieldWritesOnlyItself(t *testing.T) {
	tests := []struct {
		name  string
		input *model.LocalUIConfigSamplingInput
		want  *common.SamplingConfiguration
	}{
		{
			"dryRun",
			&model.LocalUIConfigSamplingInput{DryRun: localUiBoolPtr(true)},
			&common.SamplingConfiguration{DryRun: localUiBoolPtr(true)},
		},
		{
			"spanSamplingAttributes.disabled",
			&model.LocalUIConfigSamplingInput{SpanSamplingAttributes: &model.LocalUIConfigSpanSamplingAttributesInput{Disabled: localUiBoolPtr(true)}},
			&common.SamplingConfiguration{SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{Disabled: localUiBoolPtr(true)}},
		},
		{
			"spanSamplingAttributes.samplingCategoryDisabled",
			&model.LocalUIConfigSamplingInput{SpanSamplingAttributes: &model.LocalUIConfigSpanSamplingAttributesInput{SamplingCategoryDisabled: localUiBoolPtr(true)}},
			&common.SamplingConfiguration{SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{SamplingCategoryDisabled: localUiBoolPtr(true)}},
		},
		{
			"spanSamplingAttributes.traceDecidingRuleDisabled",
			&model.LocalUIConfigSamplingInput{SpanSamplingAttributes: &model.LocalUIConfigSpanSamplingAttributesInput{TraceDecidingRuleDisabled: localUiBoolPtr(true)}},
			&common.SamplingConfiguration{SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{TraceDecidingRuleDisabled: localUiBoolPtr(true)}},
		},
		{
			"spanSamplingAttributes.spanDecisionAttributesDisabled",
			&model.LocalUIConfigSamplingInput{SpanSamplingAttributes: &model.LocalUIConfigSpanSamplingAttributesInput{SpanDecisionAttributesDisabled: localUiBoolPtr(true)}},
			&common.SamplingConfiguration{SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{SpanDecisionAttributesDisabled: localUiBoolPtr(true)}},
		},
		{
			"tailSampling.disabled",
			&model.LocalUIConfigSamplingInput{TailSampling: &model.TailSamplingConfigInput{Disabled: localUiBoolPtr(true)}},
			&common.SamplingConfiguration{TailSampling: &commonapisampling.TailSamplingConfiguration{Disabled: localUiBoolPtr(true)}},
		},
		{
			"tailSampling.traceAggregationWaitDuration",
			&model.LocalUIConfigSamplingInput{TailSampling: &model.TailSamplingConfigInput{TraceAggregationWaitDuration: localUiStrPtr("11s")}},
			&common.SamplingConfiguration{TailSampling: &commonapisampling.TailSamplingConfiguration{TraceAggregationWaitDuration: localUiStrPtr("11s")}},
		},
		{
			"k8sHealthProbesSampling.enabled",
			&model.LocalUIConfigSamplingInput{K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{Enabled: localUiBoolPtr(true)}},
			&common.SamplingConfiguration{K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{Enabled: localUiBoolPtr(true)}},
		},
		{
			"k8sHealthProbesSampling.keepPercentage",
			&model.LocalUIConfigSamplingInput{K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfigInput{KeepPercentage: localUiFloatPtr(2.5)}},
			&common.SamplingConfiguration{K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{KeepPercentage: localUiFloatPtr(2.5)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &common.SamplingConfiguration{}

			applySamplingInput(cfg, tt.input)

			assert.Equal(t, tt.want, cfg)
		})
	}
}

func localUiConfigMap(t *testing.T, config *common.OdigosConfiguration) client.Object {
	t.Helper()

	raw, err := yaml.Marshal(config)
	require.NoError(t, err)
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosLocalUiConfigName, Namespace: localUiConfigTestNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: string(raw)},
	}
}

func storedLocalUiConfig(t *testing.T, c client.Client) *common.OdigosConfiguration {
	t.Helper()

	var cm v1.ConfigMap
	require.NoError(t, c.Get(context.Background(),
		types.NamespacedName{Namespace: localUiConfigTestNamespace, Name: consts.OdigosLocalUiConfigName}, &cm))

	config := &common.OdigosConfiguration{}
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), config))
	return config
}

// The ConfigMap is read-modify-written on every save, so a setting that was stored by an
// earlier save must survive one that does not mention it.
func TestUpdateLocalUIConfig_PreservesSettingsTheInputDoesNotMention(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := newFakeClient(newScheme(), []client.Object{
		localUiConfigMap(t, &common.OdigosConfiguration{
			ClusterName:       "prod-eu-west",
			IgnoredNamespaces: []string{"payments"},
			Rollout:           &common.RolloutConfiguration{MaxConcurrentRollouts: 7},
		}),
	})

	require.NoError(t, UpdateLocalUIConfig(context.Background(), c, model.LocalUIConfigInput{
		ComponentLogLevels: &model.LocalUIConfigComponentLogLevelsInput{Odiglet: localUiLogLevelPtr(model.OdigosLogLevelDebug)},
	}))

	stored := storedLocalUiConfig(t, c)
	assert.Equal(t, "prod-eu-west", stored.ClusterName)
	assert.Equal(t, []string{"payments"}, stored.IgnoredNamespaces)
	require.NotNil(t, stored.Rollout)
	assert.Equal(t, 7, stored.Rollout.MaxConcurrentRollouts)
	require.NotNil(t, stored.ComponentLogLevels)
	assert.Equal(t, common.LogLevelDebug, stored.ComponentLogLevels.Odiglet)
}

// The first save on a fresh install has no ConfigMap to update. The created one is owned
// by odigos-configuration so that `helm uninstall` garbage collects it.
func TestUpdateLocalUIConfig_CreatesTheConfigMapWithAnOwnerReference(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := newFakeClient(newScheme(), []client.Object{&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosConfigurationName,
			Namespace: localUiConfigTestNamespace,
			UID:       "owner-uid",
		},
	}})

	require.NoError(t, UpdateLocalUIConfig(context.Background(), c, model.LocalUIConfigInput{
		ClusterName: localUiStrPtr("prod-eu-west"),
	}))

	var cm v1.ConfigMap
	require.NoError(t, c.Get(context.Background(),
		types.NamespacedName{Namespace: localUiConfigTestNamespace, Name: consts.OdigosLocalUiConfigName}, &cm))

	require.Len(t, cm.OwnerReferences, 1)
	assert.Equal(t, consts.OdigosConfigurationName, cm.OwnerReferences[0].Name)
	assert.Equal(t, types.UID("owner-uid"), cm.OwnerReferences[0].UID)
	assert.Equal(t, "prod-eu-west", storedLocalUiConfig(t, c).ClusterName)
}

// Without the odigos-configuration ConfigMap there is nothing to own the new ConfigMap,
// and creating an orphan would leak it past uninstall.
func TestUpdateLocalUIConfig_FailsWhenTheOwnerConfigMapIsMissing(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := newFakeClient(newScheme(), nil)

	err := UpdateLocalUIConfig(context.Background(), c, model.LocalUIConfigInput{
		ClusterName: localUiStrPtr("prod-eu-west"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get odigos-configuration for owner reference")
}

func TestUpdateLocalUIConfig_RejectsAnUnparsableStoredDocument(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := newFakeClient(newScheme(), []client.Object{&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosLocalUiConfigName, Namespace: localUiConfigTestNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "clusterName: [not-a-string"},
	}})

	err := UpdateLocalUIConfig(context.Background(), c, model.LocalUIConfigInput{
		ClusterName: localUiStrPtr("prod-eu-west"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse existing config")
}

func TestApplyTraceCorrelationsInput_NilInputTouchesNothing(t *testing.T) {
	cfg := &common.OdigosConfiguration{}

	applyTraceCorrelationsInput(cfg, nil)
	applyTraceCorrelationsInput(cfg, &model.LocalUIConfigTraceCorrelationsInput{})

	assert.Equal(t, &common.OdigosConfiguration{}, cfg)
}

// An API error other than NotFound must surface rather than be mistaken for "no config
// yet", which would replace the stored document with one built from scratch.
func TestUpdateLocalUIConfig_PropagatesAReadFailure(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := fake.NewClientBuilder().
		WithScheme(newScheme()).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return apierrors.NewInternalError(errors.New("etcd is down"))
			},
		}).
		Build()

	err := UpdateLocalUIConfig(context.Background(), c, model.LocalUIConfigInput{
		ClusterName: localUiStrPtr("prod-eu-west"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "etcd is down")
}

func TestResetLocalUiConfigToFactoryDefaults_PropagatesAReadFailure(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := fake.NewClientBuilder().
		WithScheme(newScheme()).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return apierrors.NewInternalError(errors.New("etcd is down"))
			},
		}).
		Build()

	err := ResetLocalUiConfigToFactoryDefaults(context.Background(), c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "etcd is down")
}

func TestResetLocalUiConfigToFactoryDefaults_ClearsEverySetting(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := newFakeClient(newScheme(), []client.Object{localUiConfigMap(t, fullyPopulatedOverlayConfig())})

	require.NoError(t, ResetLocalUiConfigToFactoryDefaults(context.Background(), c))

	assert.Equal(t, &common.OdigosConfiguration{}, storedLocalUiConfig(t, c))
}

// Resetting an install that never saved a UI setting is already at factory defaults.
func TestResetLocalUiConfigToFactoryDefaults_NoConfigMapIsNotAnError(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, localUiConfigTestNamespace)

	c := newFakeClient(newScheme(), nil)

	require.NoError(t, ResetLocalUiConfigToFactoryDefaults(context.Background(), c))

	var cm v1.ConfigMap
	err := c.Get(context.Background(),
		types.NamespacedName{Namespace: localUiConfigTestNamespace, Name: consts.OdigosLocalUiConfigName}, &cm)
	assert.True(t, apierrors.IsNotFound(err))
}
