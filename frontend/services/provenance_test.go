package services

import (
	"context"
	"sort"
	"testing"

	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// The provenance map answers "which ConfigMap is this setting coming from?" for every
// row of the odigos settings screen. It is a hand-maintained mirror of the scheduler's
// merge order, so a field that is added to common.OdigosConfiguration and forgotten here
// silently renders as if it came from the helm baseline, and a mistyped key renders the
// same way. Neither failure mode produces an error anywhere.

const provenanceTestNamespace = "odigos-provenance-test"

func provenanceBoolPtr(v bool) *bool        { return &v }
func provenanceFloatPtr(v float64) *float64 { return &v }
func provenanceStrPtr(v string) *string     { return &v }

func overlayProvenanceKeys(config *common.OdigosConfiguration, sourceName string) []string {
	provenance := map[string]string{}
	recordOverlayProvenance(config, provenance, sourceName)

	keys := make([]string, 0, len(provenance))
	for k, v := range provenance {
		if v != sourceName {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func provenanceConfigMap(t *testing.T, name string, config *common.OdigosConfiguration) client.Object {
	t.Helper()

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: provenanceTestNamespace},
	}
	if config != nil {
		raw, err := yaml.Marshal(config)
		require.NoError(t, err)
		cm.Data = map[string]string{consts.OdigosConfigurationFileName: string(raw)}
	}
	return cm
}

// FullyPopulatedOverlayConfig sets every field an overlay ConfigMap (remote or local UI)
// is allowed to carry. Values are deliberately non-default so a "this field was never
// written" bug cannot pass by coincidence.
func fullyPopulatedOverlayConfig() *common.OdigosConfiguration {
	envInjection := common.PodManifestEnvInjectionMethod
	return &common.OdigosConfiguration{
		TelemetryEnabled:                 true,
		IgnoredNamespaces:                []string{"payments"},
		IgnoredContainers:                []string{"istio-proxy"},
		IgnoreOdigosNamespace:            provenanceBoolPtr(false),
		ClusterName:                      "prod-eu-west",
		AgentEnvVarsInjectionMethod:      &envInjection,
		CheckDeviceHealthBeforeInjection: provenanceBoolPtr(true),
		AllowConcurrentAgents:            provenanceBoolPtr(true),
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: provenanceBoolPtr(true),
			MaxConcurrentRollouts:    7,
		},
		RollbackDisabled:        provenanceBoolPtr(true),
		RollbackGraceTime:       "3m",
		RollbackStabilityWindow: "9m",
		GoAutoOffsetsCron:       "0 3 * * *",
		GoAutoOffsetsMode:       "cron",
		Sampling: &common.SamplingConfiguration{
			DryRun: provenanceBoolPtr(true),
			SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{
				Disabled:                       provenanceBoolPtr(true),
				SamplingCategoryDisabled:       provenanceBoolPtr(true),
				TraceDecidingRuleDisabled:      provenanceBoolPtr(true),
				SpanDecisionAttributesDisabled: provenanceBoolPtr(true),
			},
			TailSampling: &commonapisampling.TailSamplingConfiguration{
				Disabled:                     provenanceBoolPtr(true),
				TraceAggregationWaitDuration: provenanceStrPtr("11s"),
			},
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        provenanceBoolPtr(true),
				KeepPercentage: provenanceFloatPtr(12.5),
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
				Enabled:              provenanceBoolPtr(true),
				InputSpanAttributes:  []string{"http.route"},
				OutputSpanAttributes: []string{"db.system"},
				MetricsFlushInterval: "45s",
			},
		},
	}
}

// allOverlayProvenanceKeys is the full set the overlay recorder can emit, kept as a literal
// so a renamed or dropped key shows up as a diff rather than as a silently smaller map.
var allOverlayProvenanceKeys = []string{
	"agentEnvVarsInjectionMethod",
	"allowConcurrentAgents",
	"checkDeviceHealthBeforeInjection",
	"clusterName",
	"componentLogLevels.autoscaler",
	"componentLogLevels.collector",
	"componentLogLevels.default",
	"componentLogLevels.deviceplugin",
	"componentLogLevels.instrumentor",
	"componentLogLevels.odiglet",
	"componentLogLevels.scheduler",
	"componentLogLevels.ui",
	"goAutoOffsetsCron",
	"goAutoOffsetsMode",
	"ignoreOdigosNamespace",
	"ignoredContainers",
	"ignoredNamespaces",
	"rollbackDisabled",
	"rollbackGraceTime",
	"rollbackStabilityWindow",
	"rollout.automaticRolloutDisabled",
	"rollout.maxConcurrentRollouts",
	"sampling.dryRun",
	"sampling.k8sHealthProbesSampling.enabled",
	"sampling.k8sHealthProbesSampling.keepPercentage",
	"sampling.spanSamplingAttributes.disabled",
	"sampling.spanSamplingAttributes.samplingCategoryDisabled",
	"sampling.spanSamplingAttributes.spanDecisionAttributesDisabled",
	"sampling.spanSamplingAttributes.traceDecidingRuleDisabled",
	"sampling.tailSampling.disabled",
	"sampling.tailSampling.traceAggregationWaitDuration",
	"telemetryEnabled",
	"traceCorrelations.serviceIO.enabled",
	"traceCorrelations.serviceIO.inputSpanAttributes",
	"traceCorrelations.serviceIO.metricsFlushInterval",
	"traceCorrelations.serviceIO.outputSpanAttributes",
}

func TestRecordOverlayProvenance_FullyPopulatedOverlayRecordsEveryKey(t *testing.T) {
	got := overlayProvenanceKeys(fullyPopulatedOverlayConfig(), consts.OdigosLocalUiConfigName)

	assert.Equal(t, allOverlayProvenanceKeys, got)
}

func TestRecordOverlayProvenance_NilConfigRecordsNothing(t *testing.T) {
	provenance := map[string]string{"clusterName": "pre-existing"}

	recordOverlayProvenance(nil, provenance, consts.OdigosRemoteConfigName)

	assert.Equal(t, map[string]string{"clusterName": "pre-existing"}, provenance)
}

// An empty overlay must claim nothing: every entry it writes overrides the helm baseline
// badge in the UI, so a spurious entry is a wrong answer, not a missing one.
func TestRecordOverlayProvenance_EmptyConfigRecordsNothing(t *testing.T) {
	got := overlayProvenanceKeys(&common.OdigosConfiguration{}, consts.OdigosLocalUiConfigName)

	assert.Empty(t, got)
}

// The nested blocks are only inspected when their parent pointer is set. An overlay that
// carries an empty `sampling:` / `componentLogLevels:` / `traceCorrelations:` block claims
// no field inside it.
func TestRecordOverlayProvenance_EmptyNestedBlocksRecordNothing(t *testing.T) {
	config := &common.OdigosConfiguration{
		Rollout:            &common.RolloutConfiguration{},
		Sampling:           &common.SamplingConfiguration{},
		ComponentLogLevels: &common.ComponentLogLevels{},
		TraceCorrelations:  &common.TraceCorrelationsConfiguration{},
	}

	assert.Empty(t, overlayProvenanceKeys(config, consts.OdigosLocalUiConfigName))

	config.Sampling.SpanSamplingAttributes = &commonapisampling.SpanSamplingAttributesConfiguration{}
	config.Sampling.TailSampling = &commonapisampling.TailSamplingConfiguration{}
	config.Sampling.K8sHealthProbesSampling = &common.K8sHealthProbesSamplingConfiguration{}
	config.TraceCorrelations.ServiceIO = &common.TraceCorrelationsServiceIOConfiguration{}

	assert.Empty(t, overlayProvenanceKeys(config, consts.OdigosLocalUiConfigName))
}

// Each field is recorded under its own key and no other. This is the check that a
// copy-pasted key string cannot survive: writing "rollbackGraceTime" from the
// rollbackStabilityWindow branch keeps the map the same size but points the UI at the
// wrong row.
func TestRecordOverlayProvenance_EachFieldRecordsOnlyItsOwnKey(t *testing.T) {
	envInjection := common.LoaderEnvInjectionMethod

	tests := []struct {
		name    string
		config  *common.OdigosConfiguration
		wantKey string
	}{
		{"telemetryEnabled", &common.OdigosConfiguration{TelemetryEnabled: true}, "telemetryEnabled"},
		{"ignoredNamespaces", &common.OdigosConfiguration{IgnoredNamespaces: []string{"kube-system"}}, "ignoredNamespaces"},
		{"ignoredContainers", &common.OdigosConfiguration{IgnoredContainers: []string{"sidecar"}}, "ignoredContainers"},
		{"ignoreOdigosNamespace", &common.OdigosConfiguration{IgnoreOdigosNamespace: provenanceBoolPtr(false)}, "ignoreOdigosNamespace"},
		{"clusterName", &common.OdigosConfiguration{ClusterName: "staging"}, "clusterName"},
		{"agentEnvVarsInjectionMethod", &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &envInjection}, "agentEnvVarsInjectionMethod"},
		{"checkDeviceHealthBeforeInjection", &common.OdigosConfiguration{CheckDeviceHealthBeforeInjection: provenanceBoolPtr(false)}, "checkDeviceHealthBeforeInjection"},
		{"allowConcurrentAgents", &common.OdigosConfiguration{AllowConcurrentAgents: provenanceBoolPtr(false)}, "allowConcurrentAgents"},
		{
			"rollout.automaticRolloutDisabled",
			&common.OdigosConfiguration{Rollout: &common.RolloutConfiguration{AutomaticRolloutDisabled: provenanceBoolPtr(false)}},
			"rollout.automaticRolloutDisabled",
		},
		{
			"rollout.maxConcurrentRollouts",
			&common.OdigosConfiguration{Rollout: &common.RolloutConfiguration{MaxConcurrentRollouts: 4}},
			"rollout.maxConcurrentRollouts",
		},
		{"rollbackDisabled", &common.OdigosConfiguration{RollbackDisabled: provenanceBoolPtr(false)}, "rollbackDisabled"},
		{"rollbackGraceTime", &common.OdigosConfiguration{RollbackGraceTime: "2m"}, "rollbackGraceTime"},
		{"rollbackStabilityWindow", &common.OdigosConfiguration{RollbackStabilityWindow: "8m"}, "rollbackStabilityWindow"},
		{"goAutoOffsetsCron", &common.OdigosConfiguration{GoAutoOffsetsCron: "@daily"}, "goAutoOffsetsCron"},
		{"goAutoOffsetsMode", &common.OdigosConfiguration{GoAutoOffsetsMode: "manual"}, "goAutoOffsetsMode"},
		{
			"sampling.dryRun",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{DryRun: provenanceBoolPtr(false)}},
			"sampling.dryRun",
		},
		{
			"sampling.spanSamplingAttributes.disabled",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{Disabled: provenanceBoolPtr(false)},
			}},
			"sampling.spanSamplingAttributes.disabled",
		},
		{
			"sampling.spanSamplingAttributes.samplingCategoryDisabled",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{SamplingCategoryDisabled: provenanceBoolPtr(false)},
			}},
			"sampling.spanSamplingAttributes.samplingCategoryDisabled",
		},
		{
			"sampling.spanSamplingAttributes.traceDecidingRuleDisabled",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{TraceDecidingRuleDisabled: provenanceBoolPtr(false)},
			}},
			"sampling.spanSamplingAttributes.traceDecidingRuleDisabled",
		},
		{
			"sampling.spanSamplingAttributes.spanDecisionAttributesDisabled",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{SpanDecisionAttributesDisabled: provenanceBoolPtr(false)},
			}},
			"sampling.spanSamplingAttributes.spanDecisionAttributesDisabled",
		},
		{
			"sampling.tailSampling.disabled",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				TailSampling: &commonapisampling.TailSamplingConfiguration{Disabled: provenanceBoolPtr(false)},
			}},
			"sampling.tailSampling.disabled",
		},
		{
			"sampling.tailSampling.traceAggregationWaitDuration",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				TailSampling: &commonapisampling.TailSamplingConfiguration{TraceAggregationWaitDuration: provenanceStrPtr("4s")},
			}},
			"sampling.tailSampling.traceAggregationWaitDuration",
		},
		{
			"sampling.k8sHealthProbesSampling.enabled",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{Enabled: provenanceBoolPtr(false)},
			}},
			"sampling.k8sHealthProbesSampling.enabled",
		},
		{
			"sampling.k8sHealthProbesSampling.keepPercentage",
			&common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{
				K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{KeepPercentage: provenanceFloatPtr(0.5)},
			}},
			"sampling.k8sHealthProbesSampling.keepPercentage",
		},
		{
			"componentLogLevels.default",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{Default: common.LogLevelDebug}},
			"componentLogLevels.default",
		},
		{
			"componentLogLevels.autoscaler",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{Autoscaler: common.LogLevelDebug}},
			"componentLogLevels.autoscaler",
		},
		{
			"componentLogLevels.scheduler",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{Scheduler: common.LogLevelDebug}},
			"componentLogLevels.scheduler",
		},
		{
			"componentLogLevels.instrumentor",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{Instrumentor: common.LogLevelDebug}},
			"componentLogLevels.instrumentor",
		},
		{
			"componentLogLevels.odiglet",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{Odiglet: common.LogLevelDebug}},
			"componentLogLevels.odiglet",
		},
		{
			"componentLogLevels.deviceplugin",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{Deviceplugin: common.LogLevelDebug}},
			"componentLogLevels.deviceplugin",
		},
		{
			"componentLogLevels.ui",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{UI: common.LogLevelDebug}},
			"componentLogLevels.ui",
		},
		{
			"componentLogLevels.collector",
			&common.OdigosConfiguration{ComponentLogLevels: &common.ComponentLogLevels{Collector: common.LogLevelDebug}},
			"componentLogLevels.collector",
		},
		{
			"traceCorrelations.serviceIO.enabled",
			&common.OdigosConfiguration{TraceCorrelations: &common.TraceCorrelationsConfiguration{
				ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{Enabled: provenanceBoolPtr(false)},
			}},
			"traceCorrelations.serviceIO.enabled",
		},
		{
			"traceCorrelations.serviceIO.inputSpanAttributes",
			&common.OdigosConfiguration{TraceCorrelations: &common.TraceCorrelationsConfiguration{
				ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{InputSpanAttributes: []string{"http.route"}},
			}},
			"traceCorrelations.serviceIO.inputSpanAttributes",
		},
		{
			"traceCorrelations.serviceIO.outputSpanAttributes",
			&common.OdigosConfiguration{TraceCorrelations: &common.TraceCorrelationsConfiguration{
				ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{OutputSpanAttributes: []string{"db.system"}},
			}},
			"traceCorrelations.serviceIO.outputSpanAttributes",
		},
		{
			"traceCorrelations.serviceIO.metricsFlushInterval",
			&common.OdigosConfiguration{TraceCorrelations: &common.TraceCorrelationsConfiguration{
				ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{MetricsFlushInterval: "17s"},
			}},
			"traceCorrelations.serviceIO.metricsFlushInterval",
		},
	}

	// Every key the recorder knows about must be exercised, so a new field cannot be added
	// to recordOverlayProvenance without also being pinned here.
	covered := make([]string, 0, len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := overlayProvenanceKeys(tt.config, consts.OdigosRemoteConfigName)
			assert.Equal(t, []string{tt.wantKey}, got)
		})
		covered = append(covered, tt.wantKey)
	}
	sort.Strings(covered)
	assert.Equal(t, allOverlayProvenanceKeys, covered)
}

// telemetryEnabled is the one non-pointer field the overlay recorder inspects, so an
// overlay that explicitly turns telemetry off is indistinguishable from one that does not
// mention it and keeps the helm baseline badge.
func TestRecordOverlayProvenance_TelemetryDisabledIsNotAttributedToTheOverlay(t *testing.T) {
	got := overlayProvenanceKeys(&common.OdigosConfiguration{TelemetryEnabled: false}, consts.OdigosLocalUiConfigName)

	assert.Empty(t, got)
}

// ComputeProvenance applies remote first and then local, so a field set in both must be
// attributed to the local UI ConfigMap - that is the one the user last edited.
func TestRecordOverlayProvenance_LaterSourceWinsForTheSameField(t *testing.T) {
	provenance := map[string]string{}
	remote := &common.OdigosConfiguration{ClusterName: "from-remote", GoAutoOffsetsCron: "@hourly"}
	local := &common.OdigosConfiguration{ClusterName: "from-local"}

	recordOverlayProvenance(remote, provenance, consts.OdigosRemoteConfigName)
	recordOverlayProvenance(local, provenance, consts.OdigosLocalUiConfigName)

	assert.Equal(t, consts.OdigosLocalUiConfigName, provenance["clusterName"])
	assert.Equal(t, consts.OdigosRemoteConfigName, provenance["goAutoOffsetsCron"])
}

func TestDetectProfileProvenance_NilBaseOrEffectiveIsANoop(t *testing.T) {
	provenance := map[string]string{}

	detectProfileProvenance(nil, nil, nil, &common.OdigosConfiguration{ClusterName: "x"}, provenance)
	assert.Empty(t, provenance)

	detectProfileProvenance(&common.OdigosConfiguration{}, nil, nil, nil, provenance)
	assert.Empty(t, provenance)
}

// A profile is detected by the effective value differing from what base + overlays would
// have produced. When nothing moved the value, nothing may be attributed to a profile.
func TestDetectProfileProvenance_UnchangedEffectiveConfigDetectsNoProfile(t *testing.T) {
	withMetricsSources := func() *common.OdigosConfiguration {
		config := fullyPopulatedOverlayConfig()
		config.MetricsSources = &common.MetricsSourceConfiguration{
			HostMetrics: &common.MetricsSourceHostMetricsConfiguration{Interval: "30s"},
		}
		return config
	}
	base := withMetricsSources()
	effective := withMetricsSources()
	provenance := map[string]string{}

	detectProfileProvenance(base, nil, nil, effective, provenance)

	assert.Empty(t, provenance)
}

func TestDetectProfileProvenance_DetectsEachProfileModifiableField(t *testing.T) {
	hostPath := common.K8sHostPathMountMethod
	podManifest := common.PodManifestEnvInjectionMethod

	tests := []struct {
		name      string
		base      *common.OdigosConfiguration
		effective *common.OdigosConfiguration
		wantKey   string
	}{
		{
			name:      "rollbackDisabled",
			base:      &common.OdigosConfiguration{},
			effective: &common.OdigosConfiguration{RollbackDisabled: provenanceBoolPtr(true)},
			wantKey:   "rollbackDisabled",
		},
		{
			name:      "mountMethod",
			base:      &common.OdigosConfiguration{},
			effective: &common.OdigosConfiguration{MountMethod: &hostPath},
			wantKey:   "mountMethod",
		},
		{
			name:      "agentEnvVarsInjectionMethod",
			base:      &common.OdigosConfiguration{},
			effective: &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &podManifest},
			wantKey:   "agentEnvVarsInjectionMethod",
		},
		{
			name:      "allowConcurrentAgents",
			base:      &common.OdigosConfiguration{},
			effective: &common.OdigosConfiguration{AllowConcurrentAgents: provenanceBoolPtr(true)},
			wantKey:   "allowConcurrentAgents",
		},
		{
			name:      "checkDeviceHealthBeforeInjection",
			base:      &common.OdigosConfiguration{},
			effective: &common.OdigosConfiguration{CheckDeviceHealthBeforeInjection: provenanceBoolPtr(true)},
			wantKey:   "checkDeviceHealthBeforeInjection",
		},
		{
			name: "metricsSources",
			base: &common.OdigosConfiguration{},
			effective: &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
				SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{Interval: "45s"},
			}},
			wantKey: "metricsSources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provenance := map[string]string{}

			detectProfileProvenance(tt.base, nil, nil, tt.effective, provenance)

			assert.Equal(t, map[string]string{tt.wantKey: "profile"}, provenance)
		})
	}
}

// An overlay that already carries the effective value is what moved it, not a profile.
// Both overlays are consulted, with local taking precedence over remote.
func TestDetectProfileProvenance_OverlayValueIsNotAttributedToAProfile(t *testing.T) {
	podManifest := common.PodManifestEnvInjectionMethod

	tests := []struct {
		name   string
		remote *common.OdigosConfiguration
		local  *common.OdigosConfiguration
	}{
		{
			name: "remote overlay",
			remote: &common.OdigosConfiguration{
				RollbackDisabled:                 provenanceBoolPtr(true),
				AgentEnvVarsInjectionMethod:      &podManifest,
				AllowConcurrentAgents:            provenanceBoolPtr(true),
				CheckDeviceHealthBeforeInjection: provenanceBoolPtr(true),
			},
		},
		{
			name: "local overlay",
			local: &common.OdigosConfiguration{
				RollbackDisabled:                 provenanceBoolPtr(true),
				AgentEnvVarsInjectionMethod:      &podManifest,
				AllowConcurrentAgents:            provenanceBoolPtr(true),
				CheckDeviceHealthBeforeInjection: provenanceBoolPtr(true),
			},
		},
		{
			name: "local overlay wins over a conflicting remote overlay",
			remote: &common.OdigosConfiguration{
				RollbackDisabled:                 provenanceBoolPtr(false),
				AllowConcurrentAgents:            provenanceBoolPtr(false),
				CheckDeviceHealthBeforeInjection: provenanceBoolPtr(false),
			},
			local: &common.OdigosConfiguration{
				RollbackDisabled:                 provenanceBoolPtr(true),
				AgentEnvVarsInjectionMethod:      &podManifest,
				AllowConcurrentAgents:            provenanceBoolPtr(true),
				CheckDeviceHealthBeforeInjection: provenanceBoolPtr(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effective := &common.OdigosConfiguration{
				RollbackDisabled:                 provenanceBoolPtr(true),
				AgentEnvVarsInjectionMethod:      &podManifest,
				AllowConcurrentAgents:            provenanceBoolPtr(true),
				CheckDeviceHealthBeforeInjection: provenanceBoolPtr(true),
			}
			provenance := map[string]string{}

			detectProfileProvenance(&common.OdigosConfiguration{}, tt.remote, tt.local, effective, provenance)

			assert.Empty(t, provenance)
		})
	}
}

// mountMethod and agentEnvVarsInjectionMethod are compared through their resolved values
// because the controller materialises the default for a nil field. Without that, every
// cluster on defaults would report both fields as profile-set.
func TestDetectProfileProvenance_MaterialisedDefaultsAreNotAProfile(t *testing.T) {
	virtualDevice := common.K8sVirtualDeviceMountMethod
	loaderFallback := common.LoaderFallbackToPodManifestInjectionMethod
	base := &common.OdigosConfiguration{}
	effective := &common.OdigosConfiguration{
		MountMethod:                 &virtualDevice,
		AgentEnvVarsInjectionMethod: &loaderFallback,
	}
	provenance := map[string]string{}

	detectProfileProvenance(base, nil, nil, effective, provenance)

	assert.Empty(t, provenance)
}

// The inverse of the case above: a profile that clears an explicitly configured
// non-default value back to the default is still a profile-driven change.
func TestDetectProfileProvenance_ProfileResettingToTheDefaultIsDetected(t *testing.T) {
	hostPath := common.K8sHostPathMountMethod
	base := &common.OdigosConfiguration{MountMethod: &hostPath}
	provenance := map[string]string{}

	detectProfileProvenance(base, nil, nil, &common.OdigosConfiguration{}, provenance)

	assert.Equal(t, map[string]string{"mountMethod": "profile"}, provenance)
}

func TestResolvedMountMethod(t *testing.T) {
	hostPath := common.K8sHostPathMountMethod

	assert.Equal(t, common.K8sVirtualDeviceMountMethod, resolvedMountMethod(nil))
	assert.Equal(t, common.K8sHostPathMountMethod, resolvedMountMethod(&hostPath))
}

func TestResolvedEnvInjectionMethod(t *testing.T) {
	podManifest := common.PodManifestEnvInjectionMethod

	assert.Equal(t, common.LoaderFallbackToPodManifestInjectionMethod, resolvedEnvInjectionMethod(nil))
	assert.Equal(t, common.PodManifestEnvInjectionMethod, resolvedEnvInjectionMethod(&podManifest))
}

func TestComputeProvenance_NilEffectiveConfig(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, provenanceTestNamespace)

	got, err := ComputeProvenance(context.Background(), newFakeClient(newScheme(), nil), nil)

	require.NoError(t, err)
	assert.Nil(t, got)
}

// No overlay ConfigMaps at all is the state of a fresh helm install: every field must
// stay attributed to the helm baseline rather than blowing up.
func TestComputeProvenance_NoOverlayConfigMaps(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, provenanceTestNamespace)

	got, err := ComputeProvenance(context.Background(), newFakeClient(newScheme(), nil), fullyPopulatedOverlayConfig())

	require.NoError(t, err)
	assert.Empty(t, got)
}

// The end-to-end shape the settings screen consumes: helm baseline, a remote overlay and a
// local UI overlay that partly conflict, plus a profile-driven field none of them set.
func TestComputeProvenance_CombinesBaselineOverlaysAndProfiles(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, provenanceTestNamespace)

	base := &common.OdigosConfiguration{ClusterName: "from-helm"}
	remote := &common.OdigosConfiguration{
		ClusterName: "from-remote",
		Rollout:     &common.RolloutConfiguration{MaxConcurrentRollouts: 2},
	}
	local := &common.OdigosConfiguration{
		ClusterName:       "from-local",
		RollbackGraceTime: "6m",
	}
	effective := &common.OdigosConfiguration{
		ClusterName:           "from-local",
		Rollout:               &common.RolloutConfiguration{MaxConcurrentRollouts: 2},
		RollbackGraceTime:     "6m",
		AllowConcurrentAgents: provenanceBoolPtr(true),
	}

	c := newFakeClient(newScheme(), []client.Object{
		provenanceConfigMap(t, consts.OdigosConfigurationName, base),
		provenanceConfigMap(t, consts.OdigosRemoteConfigName, remote),
		provenanceConfigMap(t, consts.OdigosLocalUiConfigName, local),
	})

	got, err := ComputeProvenance(context.Background(), c, effective)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"clusterName":                   consts.OdigosLocalUiConfigName,
		"rollout.maxConcurrentRollouts": consts.OdigosRemoteConfigName,
		"rollbackGraceTime":             consts.OdigosLocalUiConfigName,
		"allowConcurrentAgents":         "profile",
	}, got)
}

// A malformed overlay ConfigMap must not take the settings screen down: the getters
// swallow the parse error and the remaining sources are still attributed.
func TestComputeProvenance_UnparsableOverlayIsSkipped(t *testing.T) {
	t.Setenv(consts.CurrentNamespaceEnvVar, provenanceTestNamespace)

	broken := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosRemoteConfigName, Namespace: provenanceTestNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "clusterName: [not-a-string"},
	}
	c := newFakeClient(newScheme(), []client.Object{
		broken,
		provenanceConfigMap(t, consts.OdigosLocalUiConfigName, &common.OdigosConfiguration{ClusterName: "from-local"}),
	})

	got, err := ComputeProvenance(context.Background(), c, &common.OdigosConfiguration{ClusterName: "from-local"})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"clusterName": consts.OdigosLocalUiConfigName}, got)
}
