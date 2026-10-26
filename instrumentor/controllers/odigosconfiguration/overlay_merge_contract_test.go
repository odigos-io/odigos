package odigosconfiguration

import (
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mergeConfigs folds the odigos-remote-config and odigos-local-ui-config overlays
// over the helm baseline to produce effective-config, which every other component
// reads. It is ~200 lines of hand-written per-field guards with nothing in the type
// system tying it to OdigosConfiguration, so a field added to the struct and wired
// into the UI is simply skipped here: the overlay ConfigMap holds the user's value,
// the settings screen attributes the field to the overlay, and the component acts
// on the helm value forever. That is exactly how a Go auto-offsets cron set from
// the UI came to be silently ignored (#5809).
//
// The two maps below make that decision explicit. Every field of
// OdigosConfiguration must appear in exactly one of them, so adding a field
// without deciding whether the overlay owns it fails
// TestMergeConfigs_EveryOdigosConfigurationFieldIsClassified.

func ocBoolPtr(b bool) *bool                                             { return &b }
func ocStrPtr(s string) *string                                          { return &s }
func ocFloatPtr(f float64) *float64                                      { return &f }
func ocInjection(m common.EnvInjectionMethod) *common.EnvInjectionMethod { return &m }

// ocOverlayMergedFields maps a top-level OdigosConfiguration field to a mutation
// that sets only that field to a value distinguishable from ocHelmBaseline's.
var ocOverlayMergedFields = map[string]func(*common.OdigosConfiguration){
	"TelemetryEnabled": func(c *common.OdigosConfiguration) { c.TelemetryEnabled = true },
	"IgnoredNamespaces": func(c *common.OdigosConfiguration) {
		c.IgnoredNamespaces = []string{"overlay-ns"}
	},
	"IgnoredContainers": func(c *common.OdigosConfiguration) {
		c.IgnoredContainers = []string{"overlay-container"}
	},
	"IgnoreOdigosNamespace": func(c *common.OdigosConfiguration) {
		c.IgnoreOdigosNamespace = ocBoolPtr(true)
	},
	"ClusterName":   func(c *common.OdigosConfiguration) { c.ClusterName = "overlay-cluster" },
	"McpAccessMode": func(c *common.OdigosConfiguration) { c.McpAccessMode = common.McpAccessModeReadWrite },
	"McpEnabled":    func(c *common.OdigosConfiguration) { c.McpEnabled = ocBoolPtr(true) },
	"AgentEnvVarsInjectionMethod": func(c *common.OdigosConfiguration) {
		c.AgentEnvVarsInjectionMethod = ocInjection(common.PodManifestEnvInjectionMethod)
	},
	"CheckDeviceHealthBeforeInjection": func(c *common.OdigosConfiguration) {
		c.CheckDeviceHealthBeforeInjection = ocBoolPtr(true)
	},
	"AllowConcurrentAgents": func(c *common.OdigosConfiguration) {
		c.AllowConcurrentAgents = ocBoolPtr(true)
	},
	"RollbackDisabled":        func(c *common.OdigosConfiguration) { c.RollbackDisabled = ocBoolPtr(true) },
	"RollbackGraceTime":       func(c *common.OdigosConfiguration) { c.RollbackGraceTime = "9m" },
	"RollbackStabilityWindow": func(c *common.OdigosConfiguration) { c.RollbackStabilityWindow = "11m" },
	"GoAutoOffsetsCron":       func(c *common.OdigosConfiguration) { c.GoAutoOffsetsCron = "7 7 * * *" },
	"GoAutoOffsetsMode":       func(c *common.OdigosConfiguration) { c.GoAutoOffsetsMode = "manual" },

	// Blocks merged field-by-field. One representative leaf each here; every leaf
	// is covered by ocOverlayMergedNestedFields below.
	"Rollout": func(c *common.OdigosConfiguration) {
		ocRollout(c).MaxConcurrentRollouts = 42
	},
	"Sampling": func(c *common.OdigosConfiguration) {
		ocSampling(c).DryRun = ocBoolPtr(true)
	},
	"ComponentLogLevels": func(c *common.OdigosConfiguration) {
		ocLogLevels(c).Default = common.LogLevelDebug
	},
	"Profiling": func(c *common.OdigosConfiguration) {
		ocProfiling(c).Enabled = ocBoolPtr(true)
	},
	"TraceCorrelations": func(c *common.OdigosConfiguration) {
		ocServiceIO(c).MetricsFlushInterval = "13s"
	},
}

// ocHelmOnlyFields lists the fields an overlay may not override, with the reason.
// "Helm-only" here means the value in effective-config always comes from the helm
// baseline, so writing the field into odigos-remote-config or
// odigos-local-ui-config has no effect.
var ocHelmOnlyFields = map[string]string{
	"ConfigVersion":                     "bumped by the chart, not a user setting",
	"OpenshiftEnabled":                  "platform detection performed at install time",
	"Psp":                               "cluster policy decided at install time",
	"ImagePrefix":                       "registry layout is an install-time decision",
	"SkipWebhookIssuerCreation":         "cert-manager topology decided at install time",
	"CollectorGateway":                  "gateway sizing is rendered by the chart",
	"CollectorNode":                     "node collector sizing is rendered by the chart",
	"Profiles":                          "merged separately by calculateEffectiveProfiles, not here",
	"UiMode":                            "chart-level UI deployment flag",
	"UiPaginationLimit":                 "chart-level UI deployment flag",
	"UiRemoteUrl":                       "chart-level UI deployment flag",
	"CentralBackendURL":                 "set by the central-proxy helm values",
	"MountMethod":                       "agent mount strategy decided at install time",
	"CustomContainerRuntimeSocketPath":  "node-level install detail",
	"UserInstrumentationEnvs":           "chart-level agent configuration",
	"NodeSelector":                      "scheduling is rendered by the chart",
	"KarpenterEnabled":                  "cluster autoscaler integration decided at install time",
	"Oidc":                              "authentication wiring held in helm values",
	"OdigletHealthProbeBindPort":        "node-level install detail",
	"ClickhouseJsonTypeEnabledProperty": "destination capability flag set by the chart",
	"ResourceSizePreset":                "sizing profile rendered by the chart",
	"MetricsSources":                    "chart-level metrics wiring",
	"AgentsInitContainerResources":      "init container sizing rendered by the chart",
	"TraceIdSuffix":                     "cluster identity assigned at install time",
	"AllowedTestConnectionHosts":        "egress allowlist held in helm values",
	"OdigosOwnTelemetryStore":           "self-telemetry backend rendered by the chart",
	"ImagePullSecrets":                  "registry credentials rendered by the chart",
	"Insights":                          "enterprise component only helm can deploy, so the flag is helm-only",

	// NOT a deliberate design decision: frontend/services/local_ui_config.go's
	// applyCardinalityControlInput writes cardinalityControl into
	// odigos-local-ui-config and provenance.go attributes those keys to the
	// overlay, but mergeConfigs has no branch for the block. Editing the live
	// traffic learning tunables in the UI therefore persists a value that never
	// reaches effective-config, while the settings screen reports it as
	// reconciled from the overlay. Same shape as #5809.
	"CardinalityControl": "KNOWN GAP: written by the local UI overlay but dropped here",
}

// ocOverlayMergedNestedFields covers the leaves of the blocks merged field by
// field. Each block is a run of near-identical guards, which is the shape where a
// copy-paste writes the wrong destination field, so every leaf gets its own row.
var ocOverlayMergedNestedFields = map[string]func(*common.OdigosConfiguration){
	"Rollout.AutomaticRolloutDisabled": func(c *common.OdigosConfiguration) {
		ocRollout(c).AutomaticRolloutDisabled = ocBoolPtr(true)
	},
	"Rollout.MaxConcurrentRollouts": func(c *common.OdigosConfiguration) {
		ocRollout(c).MaxConcurrentRollouts = 42
	},

	"Sampling.DryRun": func(c *common.OdigosConfiguration) { ocSampling(c).DryRun = ocBoolPtr(true) },
	"Sampling.SpanSamplingAttributes.Disabled": func(c *common.OdigosConfiguration) {
		ocSpanSampling(c).Disabled = ocBoolPtr(true)
	},
	"Sampling.SpanSamplingAttributes.SamplingCategoryDisabled": func(c *common.OdigosConfiguration) {
		ocSpanSampling(c).SamplingCategoryDisabled = ocBoolPtr(true)
	},
	"Sampling.SpanSamplingAttributes.TraceDecidingRuleDisabled": func(c *common.OdigosConfiguration) {
		ocSpanSampling(c).TraceDecidingRuleDisabled = ocBoolPtr(true)
	},
	"Sampling.SpanSamplingAttributes.SpanDecisionAttributesDisabled": func(c *common.OdigosConfiguration) {
		ocSpanSampling(c).SpanDecisionAttributesDisabled = ocBoolPtr(true)
	},
	"Sampling.TailSampling.Disabled": func(c *common.OdigosConfiguration) {
		ocTailSampling(c).Disabled = ocBoolPtr(true)
	},
	"Sampling.TailSampling.TraceAggregationWaitDuration": func(c *common.OdigosConfiguration) {
		ocTailSampling(c).TraceAggregationWaitDuration = ocStrPtr("17s")
	},
	"Sampling.K8sHealthProbesSampling.Enabled": func(c *common.OdigosConfiguration) {
		ocHealthProbes(c).Enabled = ocBoolPtr(true)
	},
	"Sampling.K8sHealthProbesSampling.KeepPercentage": func(c *common.OdigosConfiguration) {
		ocHealthProbes(c).KeepPercentage = ocFloatPtr(12.5)
	},

	"ComponentLogLevels.Default":      func(c *common.OdigosConfiguration) { ocLogLevels(c).Default = common.LogLevelDebug },
	"ComponentLogLevels.Autoscaler":   func(c *common.OdigosConfiguration) { ocLogLevels(c).Autoscaler = common.LogLevelDebug },
	"ComponentLogLevels.Scheduler":    func(c *common.OdigosConfiguration) { ocLogLevels(c).Scheduler = common.LogLevelDebug },
	"ComponentLogLevels.Instrumentor": func(c *common.OdigosConfiguration) { ocLogLevels(c).Instrumentor = common.LogLevelDebug },
	"ComponentLogLevels.Odiglet":      func(c *common.OdigosConfiguration) { ocLogLevels(c).Odiglet = common.LogLevelDebug },
	"ComponentLogLevels.Deviceplugin": func(c *common.OdigosConfiguration) { ocLogLevels(c).Deviceplugin = common.LogLevelDebug },
	"ComponentLogLevels.UI":           func(c *common.OdigosConfiguration) { ocLogLevels(c).UI = common.LogLevelDebug },
	"ComponentLogLevels.Collector":    func(c *common.OdigosConfiguration) { ocLogLevels(c).Collector = common.LogLevelDebug },

	"Profiling.Enabled": func(c *common.OdigosConfiguration) { ocProfiling(c).Enabled = ocBoolPtr(true) },
	"Profiling.Exporter": func(c *common.OdigosConfiguration) {
		ocProfiling(c).Exporter = &common.OtlpExporterConfiguration{Timeout: "19s"}
	},
	"Profiling.Ui.SlotTTLSeconds": func(c *common.OdigosConfiguration) { ocProfilingUi(c).SlotTTLSeconds = 61 },
	"Profiling.Ui.MaxSlots":       func(c *common.OdigosConfiguration) { ocProfilingUi(c).MaxSlots = 62 },
	"Profiling.Ui.SlotMaxBytes":   func(c *common.OdigosConfiguration) { ocProfilingUi(c).SlotMaxBytes = 63 },

	"TraceCorrelations.ServiceIO.Enabled": func(c *common.OdigosConfiguration) {
		ocServiceIO(c).Enabled = ocBoolPtr(true)
	},
	"TraceCorrelations.ServiceIO.InputSpanAttributes": func(c *common.OdigosConfiguration) {
		ocServiceIO(c).InputSpanAttributes = []string{"overlay.in"}
	},
	"TraceCorrelations.ServiceIO.OutputSpanAttributes": func(c *common.OdigosConfiguration) {
		ocServiceIO(c).OutputSpanAttributes = []string{"overlay.out"}
	},
	"TraceCorrelations.ServiceIO.MetricsFlushInterval": func(c *common.OdigosConfiguration) {
		ocServiceIO(c).MetricsFlushInterval = "13s"
	},
}

// ocUnmergedNestedFields are leaves inside an otherwise-merged block that
// mergeConfigs skips, so an overlay cannot reach them.
var ocUnmergedNestedFields = map[string]string{
	"Profiling.Symbolization": "native symbolizer wiring is rendered by the chart",
}

func ocRollout(c *common.OdigosConfiguration) *common.RolloutConfiguration {
	if c.Rollout == nil {
		c.Rollout = &common.RolloutConfiguration{}
	}
	return c.Rollout
}

func ocSampling(c *common.OdigosConfiguration) *common.SamplingConfiguration {
	if c.Sampling == nil {
		c.Sampling = &common.SamplingConfiguration{}
	}
	return c.Sampling
}

func ocSpanSampling(c *common.OdigosConfiguration) *sampling.SpanSamplingAttributesConfiguration {
	s := ocSampling(c)
	if s.SpanSamplingAttributes == nil {
		s.SpanSamplingAttributes = &sampling.SpanSamplingAttributesConfiguration{}
	}
	return s.SpanSamplingAttributes
}

func ocTailSampling(c *common.OdigosConfiguration) *sampling.TailSamplingConfiguration {
	s := ocSampling(c)
	if s.TailSampling == nil {
		s.TailSampling = &sampling.TailSamplingConfiguration{}
	}
	return s.TailSampling
}

func ocHealthProbes(c *common.OdigosConfiguration) *common.K8sHealthProbesSamplingConfiguration {
	s := ocSampling(c)
	if s.K8sHealthProbesSampling == nil {
		s.K8sHealthProbesSampling = &common.K8sHealthProbesSamplingConfiguration{}
	}
	return s.K8sHealthProbesSampling
}

func ocLogLevels(c *common.OdigosConfiguration) *common.ComponentLogLevels {
	if c.ComponentLogLevels == nil {
		c.ComponentLogLevels = &common.ComponentLogLevels{}
	}
	return c.ComponentLogLevels
}

func ocProfiling(c *common.OdigosConfiguration) *common.ProfilingConfiguration {
	if c.Profiling == nil {
		c.Profiling = &common.ProfilingConfiguration{}
	}
	return c.Profiling
}

func ocProfilingUi(c *common.OdigosConfiguration) *common.ProfilingUiConfiguration {
	p := ocProfiling(c)
	if p.Ui == nil {
		p.Ui = &common.ProfilingUiConfiguration{}
	}
	return p.Ui
}

func ocServiceIO(c *common.OdigosConfiguration) *common.TraceCorrelationsServiceIOConfiguration {
	if c.TraceCorrelations == nil {
		c.TraceCorrelations = &common.TraceCorrelationsConfiguration{}
	}
	if c.TraceCorrelations.ServiceIO == nil {
		c.TraceCorrelations.ServiceIO = &common.TraceCorrelationsServiceIOConfiguration{}
	}
	return c.TraceCorrelations.ServiceIO
}

// ocHelmBaseline is a helm baseline that already has a value for every mergeable
// field, each different from what the overlay fixtures write. Starting from a
// fully-populated base is what makes "the overlay reached the wrong sibling"
// observable: merging onto a zero-valued base cannot tell a field that landed
// correctly from one that landed next door.
func ocHelmBaseline() *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		ConfigVersion:                    3,
		TelemetryEnabled:                 false,
		IgnoredNamespaces:                []string{"kube-system"},
		IgnoredContainers:                []string{"istio-proxy"},
		IgnoreOdigosNamespace:            ocBoolPtr(false),
		ClusterName:                      "helm-cluster",
		McpAccessMode:                    common.McpAccessModeReadOnly,
		McpEnabled:                       ocBoolPtr(false),
		AgentEnvVarsInjectionMethod:      ocInjection(common.LoaderEnvInjectionMethod),
		CheckDeviceHealthBeforeInjection: ocBoolPtr(false),
		AllowConcurrentAgents:            ocBoolPtr(false),
		RollbackDisabled:                 ocBoolPtr(false),
		RollbackGraceTime:                "5m",
		RollbackStabilityWindow:          "6m",
		GoAutoOffsetsCron:                "0 0 * * *",
		GoAutoOffsetsMode:                "cron",
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: ocBoolPtr(false),
			MaxConcurrentRollouts:    4,
		},
		Sampling: &common.SamplingConfiguration{
			DryRun: ocBoolPtr(false),
			SpanSamplingAttributes: &sampling.SpanSamplingAttributesConfiguration{
				Disabled:                       ocBoolPtr(false),
				SamplingCategoryDisabled:       ocBoolPtr(false),
				TraceDecidingRuleDisabled:      ocBoolPtr(false),
				SpanDecisionAttributesDisabled: ocBoolPtr(false),
			},
			TailSampling: &sampling.TailSamplingConfiguration{
				Disabled:                     ocBoolPtr(false),
				TraceAggregationWaitDuration: ocStrPtr("3s"),
			},
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        ocBoolPtr(false),
				KeepPercentage: ocFloatPtr(1.5),
			},
		},
		ComponentLogLevels: &common.ComponentLogLevels{
			Default:      common.LogLevelInfo,
			Autoscaler:   common.LogLevelInfo,
			Scheduler:    common.LogLevelInfo,
			Instrumentor: common.LogLevelInfo,
			Odiglet:      common.LogLevelInfo,
			Deviceplugin: common.LogLevelInfo,
			UI:           common.LogLevelInfo,
			Collector:    common.LogLevelInfo,
		},
		Profiling: &common.ProfilingConfiguration{
			Enabled:  ocBoolPtr(false),
			Exporter: &common.OtlpExporterConfiguration{Timeout: "8s"},
			Ui: &common.ProfilingUiConfiguration{
				SlotTTLSeconds: 11,
				MaxSlots:       12,
				SlotMaxBytes:   13,
			},
		},
		TraceCorrelations: &common.TraceCorrelationsConfiguration{
			ServiceIO: &common.TraceCorrelationsServiceIOConfiguration{
				Enabled:              ocBoolPtr(false),
				InputSpanAttributes:  []string{"helm.in"},
				OutputSpanAttributes: []string{"helm.out"},
				MetricsFlushInterval: "7s",
			},
		},
	}
}

// ocAssertMergesOnlyItself is the whole-struct form of "this field landed".
// Asserting just the destination field cannot see a guard that also overwrote a
// sibling, and asserting a hand-written expectation cannot see a field dropped
// from both the fixture and the expectation - so the expectation is derived by
// applying the very same mutation to a second copy of the baseline.
func ocAssertMergesOnlyItself(t *testing.T, set func(*common.OdigosConfiguration)) {
	t.Helper()

	want := ocHelmBaseline()
	set(want)
	require.NotEqual(t, ocHelmBaseline(), want,
		"the fixture must differ from the baseline, otherwise a merge that does nothing passes")

	overlay := &common.OdigosConfiguration{}
	set(overlay)

	got := ocHelmBaseline()
	mergeConfigs(got, overlay)

	require.Equal(t, want, got)
}

func TestMergeConfigs_EachOverlayFieldOverridesOnlyItself(t *testing.T) {
	for name, set := range ocOverlayMergedFields {
		t.Run(name, func(t *testing.T) { ocAssertMergesOnlyItself(t, set) })
	}
}

func TestMergeConfigs_EachNestedOverlayFieldOverridesOnlyItself(t *testing.T) {
	for name, set := range ocOverlayMergedNestedFields {
		t.Run(name, func(t *testing.T) { ocAssertMergesOnlyItself(t, set) })
	}
}

// An overlay that carries nothing must be byte-identical to the baseline. Without
// this row every per-field assertion above could be satisfied by a merge that
// copies the whole overlay struct over the base.
func TestMergeConfigs_EmptyOverlayIsAnExactNoop(t *testing.T) {
	got := ocHelmBaseline()
	mergeConfigs(got, &common.OdigosConfiguration{})
	require.Equal(t, ocHelmBaseline(), got)

	mergeConfigs(got, nil)
	require.Equal(t, ocHelmBaseline(), got)
}

// A field added to OdigosConfiguration and wired into the UI, but not added to
// mergeConfigs, is persisted into the overlay ConfigMap and then silently
// dropped. Nothing else in the build catches that, so the decision is recorded
// here per field and this test fails until a new field is classified.
func TestMergeConfigs_EveryOdigosConfigurationFieldIsClassified(t *testing.T) {
	ty := reflect.TypeOf(common.OdigosConfiguration{})
	require.Positive(t, ty.NumField())

	for i := 0; i < ty.NumField(); i++ {
		name := ty.Field(i).Name
		_, merged := ocOverlayMergedFields[name]
		reason, helmOnly := ocHelmOnlyFields[name]

		assert.NotEqual(t, merged, helmOnly,
			"OdigosConfiguration.%s must be listed in exactly one of ocOverlayMergedFields "+
				"(the overlay can override it) or ocHelmOnlyFields (it cannot)", name)
		if helmOnly {
			assert.NotEmpty(t, reason, "ocHelmOnlyFields[%q] needs a reason", name)
		}
	}

	assert.Equal(t, ty.NumField(), len(ocOverlayMergedFields)+len(ocHelmOnlyFields),
		"the classification maps name a field that OdigosConfiguration no longer has")
}

// Same gate one level down: the five blocks merged leaf-by-leaf grow new leaves
// over time, and a leaf nobody added a guard for behaves like a whole unmerged
// field while looking covered because its siblings are.
func TestMergeConfigs_EveryLeafOfAMergedBlockIsClassified(t *testing.T) {
	blocks := map[string]any{
		"Rollout":                          common.RolloutConfiguration{},
		"Sampling":                         common.SamplingConfiguration{},
		"Sampling.SpanSamplingAttributes":  sampling.SpanSamplingAttributesConfiguration{},
		"Sampling.TailSampling":            sampling.TailSamplingConfiguration{},
		"Sampling.K8sHealthProbesSampling": common.K8sHealthProbesSamplingConfiguration{},
		"ComponentLogLevels":               common.ComponentLogLevels{},
		"Profiling":                        common.ProfilingConfiguration{},
		"Profiling.Ui":                     common.ProfilingUiConfiguration{},
		"TraceCorrelations":                common.TraceCorrelationsConfiguration{},
		"TraceCorrelations.ServiceIO":      common.TraceCorrelationsServiceIOConfiguration{},
	}

	// A leaf is classified if it has its own row, is itself one of the blocks
	// walked here, or is explicitly recorded as unmerged.
	classified := 0
	for prefix, block := range blocks {
		ty := reflect.TypeOf(block)
		require.Positive(t, ty.NumField(), "block %s has no fields", prefix)

		for i := 0; i < ty.NumField(); i++ {
			path := prefix + "." + ty.Field(i).Name
			_, merged := ocOverlayMergedNestedFields[path]
			_, unmerged := ocUnmergedNestedFields[path]
			_, isBlock := blocks[path]

			assert.True(t, merged || unmerged || isBlock,
				"%s is a leaf of a block mergeConfigs merges field-by-field, but it has no guard "+
					"and is not recorded in ocUnmergedNestedFields - an overlay cannot reach it", path)
			if merged {
				classified++
			}
		}
	}
	assert.Equal(t, len(ocOverlayMergedNestedFields), classified,
		"ocOverlayMergedNestedFields names a leaf that no longer exists")
}

// The block guards are `if overlay.Block != nil`, so an overlay that allocates a
// block without setting any leaf must change nothing. This is the shape the local
// UI overlay writers produce when they create the nested structs before deciding
// what to write, and it is what keeps a UI visit from resetting helm values.
func TestMergeConfigs_AllocatedButEmptyOverlayBlocksChangeNothing(t *testing.T) {
	for name, allocate := range map[string]func(*common.OdigosConfiguration){
		"Rollout":                          func(c *common.OdigosConfiguration) { ocRollout(c) },
		"Sampling":                         func(c *common.OdigosConfiguration) { ocSampling(c) },
		"Sampling.SpanSamplingAttributes":  func(c *common.OdigosConfiguration) { ocSpanSampling(c) },
		"Sampling.TailSampling":            func(c *common.OdigosConfiguration) { ocTailSampling(c) },
		"Sampling.K8sHealthProbesSampling": func(c *common.OdigosConfiguration) { ocHealthProbes(c) },
		"ComponentLogLevels":               func(c *common.OdigosConfiguration) { ocLogLevels(c) },
		"Profiling":                        func(c *common.OdigosConfiguration) { ocProfiling(c) },
		"Profiling.Ui":                     func(c *common.OdigosConfiguration) { ocProfilingUi(c) },
		"TraceCorrelations.ServiceIO":      func(c *common.OdigosConfiguration) { ocServiceIO(c) },
	} {
		t.Run(name, func(t *testing.T) {
			overlay := &common.OdigosConfiguration{}
			allocate(overlay)

			got := ocHelmBaseline()
			mergeConfigs(got, overlay)
			require.Equal(t, ocHelmBaseline(), got)
		})
	}
}

// mergeConfigs is called twice in a row by the controller - remote config first,
// then the local UI overlay - so the second call must win on every field the two
// overlays share, and must not undo the first on the fields only it set.
func TestMergeConfigs_LaterOverlayWinsPerField(t *testing.T) {
	base := ocHelmBaseline()

	remote := &common.OdigosConfiguration{
		ClusterName:       "from-remote",
		RollbackGraceTime: "30m",
	}
	localUI := &common.OdigosConfiguration{
		ClusterName:             "from-local-ui",
		RollbackStabilityWindow: "40m",
	}

	mergeConfigs(base, remote)
	mergeConfigs(base, localUI)

	assert.Equal(t, "from-local-ui", base.ClusterName, "the local UI overlay has the last word")
	assert.Equal(t, "30m", base.RollbackGraceTime,
		"a field only the remote overlay set must survive the local UI pass")
	assert.Equal(t, "40m", base.RollbackStabilityWindow)
}

// TelemetryEnabled is the one merged field whose guard is the value itself
// (`if overlay.TelemetryEnabled`) rather than a nil or empty check, so it is
// one-way: an overlay can opt in but can never opt back out.
func TestMergeConfigs_TelemetryEnabledCannotBeTurnedOffByAnOverlay(t *testing.T) {
	base := ocHelmBaseline()
	base.TelemetryEnabled = true

	mergeConfigs(base, &common.OdigosConfiguration{TelemetryEnabled: false})
	assert.True(t, base.TelemetryEnabled,
		"a false bool is indistinguishable from an absent one in this struct, so it cannot clear the baseline")
}
