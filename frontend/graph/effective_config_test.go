package graph

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EffectiveConfigToModel is the read half of the odigos settings screen: it flattens the
// merged odigos-effective-config document into the GraphQL model and, for every value it
// surfaces, records which ConfigMap the value came from. Both halves are hand-maintained
// per-field lists, so a field that is added, renamed or removed on one side and not the
// other produces a blank row or a wrong "reconciled from" badge - never an error.

func effBoolPtr(v bool) *bool        { return &v }
func effIntPtr(v int) *int           { return &v }
func effStrPtr(v string) *string     { return &v }
func effFloatPtr(v float64) *float64 { return &v }

// fullyPopulatedOdigosConfig sets every field of common.OdigosConfiguration that the
// effective config model is supposed to surface, with values that are distinguishable
// from the zero value so a dropped assignment cannot pass by coincidence.
func fullyPopulatedOdigosConfig() *common.OdigosConfiguration {
	mountMethod := common.K8sHostPathMountMethod
	envInjection := common.PodManifestEnvInjectionMethod
	gatewayNodeSelector := map[string]string{"gateway": "yes"}
	mcpAccessMode := common.McpAccessMode("read-only")
	spanMetricsMode := commonapisampling.SpanMetricsMode("all")

	return &common.OdigosConfiguration{
		ConfigVersion:                     7,
		TelemetryEnabled:                  true,
		OpenshiftEnabled:                  true,
		IgnoredNamespaces:                 []string{"payments"},
		IgnoredContainers:                 []string{"istio-proxy"},
		IgnoreOdigosNamespace:             effBoolPtr(true),
		Psp:                               true,
		ImagePrefix:                       "registry.example.com",
		SkipWebhookIssuerCreation:         true,
		Profiles:                          []common.ProfileName{"size_m", "db-payloads"},
		AllowConcurrentAgents:             effBoolPtr(true),
		UiMode:                            common.UiModeReadonly,
		UiPaginationLimit:                 42,
		UiRemoteUrl:                       "https://ui.example.com",
		McpAccessMode:                     mcpAccessMode,
		McpEnabled:                        effBoolPtr(true),
		CentralBackendURL:                 "https://central.example.com",
		ClusterName:                       "prod-eu-west",
		MountMethod:                       &mountMethod,
		CustomContainerRuntimeSocketPath:  "/run/crio/crio.sock",
		AgentEnvVarsInjectionMethod:       &envInjection,
		NodeSelector:                      map[string]string{"odigos": "enabled"},
		KarpenterEnabled:                  effBoolPtr(true),
		RollbackDisabled:                  effBoolPtr(true),
		RollbackGraceTime:                 "3m",
		RollbackStabilityWindow:           "9m",
		OdigletHealthProbeBindPort:        8899,
		GoAutoOffsetsCron:                 "0 3 * * *",
		GoAutoOffsetsMode:                 "cron",
		ClickhouseJsonTypeEnabledProperty: effBoolPtr(true),
		CheckDeviceHealthBeforeInjection:  effBoolPtr(true),
		ResourceSizePreset:                "size_l",
		TraceIdSuffix:                     "A3",
		AllowedTestConnectionHosts:        []string{"jaeger.example.com"},
		ImagePullSecrets:                  []string{"registry-creds"},
		Insights:                          &common.InsightsConfiguration{Enabled: effBoolPtr(true)},

		UserInstrumentationEnvs: &common.UserInstrumentationEnvs{
			Languages: map[common.ProgrammingLanguage]common.LanguageConfig{
				common.JavaProgrammingLanguage: {Enabled: true, EnvVars: map[string]string{"JAVA_TOOL_OPTIONS": "-Xmx1g"}},
			},
		},
		Rollout: &common.RolloutConfiguration{
			AutomaticRolloutDisabled: effBoolPtr(true),
			MaxConcurrentRollouts:    7,
		},
		Oidc: &common.OidcConfiguration{
			TenantUrl:    "https://abc-123.okta.com",
			ClientId:     "client-id",
			ClientSecret: "client-secret",
		},
		AgentsInitContainerResources: &common.AgentsInitContainerResources{
			RequestCPUm:      11,
			LimitCPUm:        22,
			RequestMemoryMiB: 33,
			LimitMemoryMiB:   44,
		},
		OdigosOwnTelemetryStore: &common.OdigosOwnTelemetryConfiguration{
			MetricsStoreDisabled: effBoolPtr(true),
		},
		Profiling: &common.ProfilingConfiguration{Enabled: effBoolPtr(true)},
		CollectorGateway: &common.CollectorGatewayConfiguration{
			MinReplicas:                1,
			MaxReplicas:                9,
			RequestMemoryMiB:           101,
			LimitMemoryMiB:             202,
			RequestCPUm:                303,
			LimitCPUm:                  404,
			MemoryLimiterLimitMiB:      505,
			MemoryLimiterSpikeLimitMiB: 606,
			GoMemLimitMib:              707,
			ClusterMetricsEnabled:      effBoolPtr(true),
			HttpsProxyAddress:          effStrPtr("https://proxy.example.com:3128"),
			ServiceGraph:               &common.ServiceGraphOptions{Disabled: effBoolPtr(true)},
			NodeSelector:               &gatewayNodeSelector,
			DeploymentName:             "custom-gateway",
		},
		CollectorNode: &common.CollectorNodeConfiguration{
			CollectorOwnMetricsPort:    55682,
			RequestMemoryMiB:           111,
			LimitMemoryMiB:             222,
			RequestCPUm:                333,
			LimitCPUm:                  444,
			MemoryLimiterLimitMiB:      555,
			MemoryLimiterSpikeLimitMiB: 666,
			GoMemLimitMib:              777,
			EnableDataCompression:      effBoolPtr(true),
			OtlpExporterConfiguration: &common.OtlpExporterConfiguration{
				EnableDataCompression: effBoolPtr(true),
				Timeout:               "13s",
				RetryOnFailure: &common.RetryOnFailure{
					Enabled:         effBoolPtr(true),
					InitialInterval: "1s",
					MaxInterval:     "2s",
					MaxElapsedTime:  "3s",
				},
			},
		},
		MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{
				Disabled:                     effBoolPtr(true),
				Interval:                     "61s",
				MetricsExpiration:            "6m",
				AdditionalDimensions:         []string{"http.method"},
				HistogramDisabled:            true,
				ExplicitHistogramBuckets:     []string{"1ms", "2ms"},
				IncludedProcessInDimensions:  effBoolPtr(true),
				ExcludedResourceAttributes:   []string{"process.pid"},
				ResourceMetricsKeyAttributes: []string{"service.name"},
				SpanMetricsMode:              &spanMetricsMode,
			},
			HostMetrics:      &common.MetricsSourceHostMetricsConfiguration{Disabled: effBoolPtr(true), Interval: "31s"},
			KubeletStats:     &common.MetricsSourceKubeletStatsConfiguration{Disabled: effBoolPtr(true), Interval: "32s"},
			OdigosOwnMetrics: &common.MetricsSourceOdigosOwnMetricsConfiguration{Interval: "33s"},
			AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{
				SpanMetrics: &common.MetricsSourceAgentSpanMetricsConfiguration{Enabled: true},
				RuntimeMetrics: &common.MetricsSourceAgentRuntimeMetricsConfiguration{
					Java: &common.MetricsSourceAgentJavaRuntimeMetricsConfiguration{
						Disabled: effBoolPtr(true),
						Metrics: []common.MetricsSourceAgentRuntimeMetricConfiguration{
							{Name: "jvm.memory.used", Disabled: effBoolPtr(true)},
							{Name: "jvm.gc.duration"},
						},
					},
				},
			},
		},
		Sampling: &common.SamplingConfiguration{
			DryRun: effBoolPtr(true),
			SpanSamplingAttributes: &commonapisampling.SpanSamplingAttributesConfiguration{
				Disabled:                       effBoolPtr(true),
				SamplingCategoryDisabled:       effBoolPtr(true),
				TraceDecidingRuleDisabled:      effBoolPtr(true),
				SpanDecisionAttributesDisabled: effBoolPtr(true),
			},
			TailSampling: &commonapisampling.TailSamplingConfiguration{
				Disabled:                     effBoolPtr(true),
				TraceAggregationWaitDuration: effStrPtr("11s"),
			},
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        effBoolPtr(true),
				KeepPercentage: effFloatPtr(12.5),
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
				Enabled:              effBoolPtr(true),
				InputSpanAttributes:  []string{"http.route"},
				OutputSpanAttributes: []string{"db.system"},
				MetricsFlushInterval: "45s",
			},
		},
	}
}

func TestEffectiveConfigToModel_NilConfig(t *testing.T) {
	got, err := EffectiveConfigToModel(nil, map[string]string{"clusterName": "x"})

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestEffectiveConfigToModel_SurfacesEveryScalarField(t *testing.T) {
	config := fullyPopulatedOdigosConfig()

	got, err := EffectiveConfigToModel(config, nil)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 7, got.ConfigVersion)
	assert.Equal(t, effBoolPtr(true), got.TelemetryEnabled)
	assert.Equal(t, effBoolPtr(true), got.OpenshiftEnabled)
	assert.Equal(t, effBoolPtr(true), got.Psp)
	assert.Equal(t, effBoolPtr(true), got.SkipWebhookIssuerCreation)
	assert.Equal(t, effBoolPtr(true), got.IgnoreOdigosNamespace)
	assert.Equal(t, effBoolPtr(true), got.ClickhouseJSONTypeEnabled)
	assert.Equal(t, effStrPtr("registry.example.com"), got.ImagePrefix)
	assert.Equal(t, effStrPtr("https://ui.example.com"), got.UIRemoteURL)
	assert.Equal(t, effStrPtr("https://central.example.com"), got.CentralBackendURL)
	assert.Equal(t, effStrPtr("prod-eu-west"), got.ClusterName)
	assert.Equal(t, effStrPtr("/run/crio/crio.sock"), got.CustomContainerRuntimeSocketPath)
	assert.Equal(t, effStrPtr("0 3 * * *"), got.GoAutoOffsetsCron)
	assert.Equal(t, effStrPtr("cron"), got.GoAutoOffsetsMode)
	assert.Equal(t, effStrPtr("size_l"), got.ResourceSizePreset)
	assert.Equal(t, effStrPtr("A3"), got.TraceIDSuffix)
	assert.Equal(t, effIntPtr(42), got.UIPaginationLimit)
	assert.Equal(t, effIntPtr(8899), got.OdigletHealthProbeBindPort)
	assert.Equal(t, []string{"payments"}, got.IgnoredNamespaces)
	assert.Equal(t, []string{"istio-proxy"}, got.IgnoredContainers)
	assert.Equal(t, []string{"jaeger.example.com"}, got.AllowedTestConnectionHosts)
	assert.Equal(t, []string{"registry-creds"}, got.ImagePullSecrets)
	assert.Equal(t, []string{"size_m", "db-payloads"}, got.Profiles)
	assert.Equal(t, effStrPtr(`{"odigos":"enabled"}`), got.NodeSelector)

	require.NotNil(t, got.UIMode)
	assert.Equal(t, model.UIModeReadonly, *got.UIMode)

	require.NotNil(t, got.AllowConcurrentAgents)
	assert.Equal(t, effBoolPtr(true), got.AllowConcurrentAgents.Enabled)
	require.NotNil(t, got.Karpenter)
	assert.Equal(t, effBoolPtr(true), got.Karpenter.Enabled)

	require.NotNil(t, got.Instrumentor)
	assert.Equal(t, effStrPtr(string(common.K8sHostPathMountMethod)), got.Instrumentor.MountMethod)
	assert.Equal(t, effStrPtr(string(common.PodManifestEnvInjectionMethod)), got.Instrumentor.AgentEnvVarsInjectionMethod)
	assert.Equal(t, effBoolPtr(true), got.Instrumentor.CheckDeviceHealthBeforeInjection)

	require.NotNil(t, got.Rollout)
	assert.Equal(t, effBoolPtr(true), got.Rollout.AutomaticRolloutDisabled)
	assert.Equal(t, effIntPtr(7), got.Rollout.MaxConcurrentRollouts)

	require.NotNil(t, got.AutoRollback)
	assert.Equal(t, effBoolPtr(true), got.AutoRollback.Disabled)
	assert.Equal(t, effStrPtr("3m"), got.AutoRollback.GraceTime)
	assert.Equal(t, effStrPtr("9m"), got.AutoRollback.StabilityWindowTime)

	require.NotNil(t, got.Oidc)
	assert.Equal(t, effStrPtr("https://abc-123.okta.com"), got.Oidc.TenantURL)
	assert.Equal(t, effStrPtr("client-id"), got.Oidc.ClientID)
	assert.Equal(t, effStrPtr("client-secret"), got.Oidc.ClientSecret)

	require.NotNil(t, got.AgentsInitContainerResources)
	assert.Equal(t, effIntPtr(11), got.AgentsInitContainerResources.RequestCPUm)
	assert.Equal(t, effIntPtr(22), got.AgentsInitContainerResources.LimitCPUm)
	assert.Equal(t, effIntPtr(33), got.AgentsInitContainerResources.RequestMemoryMiB)
	assert.Equal(t, effIntPtr(44), got.AgentsInitContainerResources.LimitMemoryMiB)

	require.NotNil(t, got.OdigosOwnTelemetryStore)
	assert.Equal(t, effBoolPtr(true), got.OdigosOwnTelemetryStore.MetricsStoreDisabled)

	require.NotNil(t, got.Profiling)
	assert.Equal(t, effBoolPtr(true), got.Profiling.Enabled)

	require.NotNil(t, got.UserInstrumentationEnvs)
	require.NotNil(t, got.UserInstrumentationEnvs.Languages)
	assert.JSONEq(t, `{"java":{"enabled":true,"env":{"JAVA_TOOL_OPTIONS":"-Xmx1g"}}}`,
		*got.UserInstrumentationEnvs.Languages)

	require.NotNil(t, got.TraceCorrelations)
	require.NotNil(t, got.TraceCorrelations.ServiceIo)
	assert.Equal(t, effBoolPtr(true), got.TraceCorrelations.ServiceIo.Enabled)
	assert.Equal(t, []string{"http.route"}, got.TraceCorrelations.ServiceIo.InputSpanAttributes)
	assert.Equal(t, []string{"db.system"}, got.TraceCorrelations.ServiceIo.OutputSpanAttributes)
	assert.Equal(t, effStrPtr("45s"), got.TraceCorrelations.ServiceIo.MetricsFlushInterval)
}

func TestEffectiveConfigToModel_SurfacesTheCollectorConfigs(t *testing.T) {
	got, err := EffectiveConfigToModel(fullyPopulatedOdigosConfig(), nil)
	require.NoError(t, err)

	require.NotNil(t, got.CollectorGateway)
	assert.Equal(t, &model.CollectorGatewayConfig{
		MinReplicas:                effIntPtr(1),
		MaxReplicas:                effIntPtr(9),
		RequestMemoryMiB:           effIntPtr(101),
		LimitMemoryMiB:             effIntPtr(202),
		RequestCPUm:                effIntPtr(303),
		LimitCPUm:                  effIntPtr(404),
		MemoryLimiterLimitMiB:      effIntPtr(505),
		MemoryLimiterSpikeLimitMiB: effIntPtr(606),
		GoMemLimitMiB:              effIntPtr(707),
		ClusterMetricsEnabled:      effBoolPtr(true),
		HTTPSProxyAddress:          effStrPtr("https://proxy.example.com:3128"),
		ServiceGraphDisabled:       effBoolPtr(true),
		NodeSelector:               effStrPtr(`{"gateway":"yes"}`),
	}, got.CollectorGateway)

	require.NotNil(t, got.CollectorNode)
	assert.Equal(t, &model.CollectorNodeConfig{
		CollectorOwnMetricsPort:    effIntPtr(55682),
		RequestMemoryMiB:           effIntPtr(111),
		LimitMemoryMiB:             effIntPtr(222),
		RequestCPUm:                effIntPtr(333),
		LimitCPUm:                  effIntPtr(444),
		MemoryLimiterLimitMiB:      effIntPtr(555),
		MemoryLimiterSpikeLimitMiB: effIntPtr(666),
		GoMemLimitMiB:              effIntPtr(777),
		EnableDataCompression:      effBoolPtr(true),
		OtlpExporterConfiguration: &model.OtlpExporterConfig{
			EnableDataCompression: effBoolPtr(true),
			Timeout:               effStrPtr("13s"),
			RetryOnFailure: &model.RetryOnFailureConfig{
				Enabled:         effBoolPtr(true),
				InitialInterval: effStrPtr("1s"),
				MaxInterval:     effStrPtr("2s"),
				MaxElapsedTime:  effStrPtr("3s"),
			},
		},
	}, got.CollectorNode)
}

func TestEffectiveConfigToModel_SurfacesTheMetricsSources(t *testing.T) {
	got, err := EffectiveConfigToModel(fullyPopulatedOdigosConfig(), nil)
	require.NoError(t, err)

	require.NotNil(t, got.MetricsSources)
	assert.Equal(t, &model.MetricsSourceSpanMetricsConfig{
		Disabled:                     effBoolPtr(true),
		Interval:                     effStrPtr("61s"),
		MetricsExpiration:            effStrPtr("6m"),
		AdditionalDimensions:         []string{"http.method"},
		HistogramBuckets:             []string{"1ms", "2ms"},
		IncludedProcessInDimensions:  effBoolPtr(true),
		ExcludedResourceAttributes:   []string{"process.pid"},
		ResourceMetricsKeyAttributes: []string{"service.name"},
		HistogramDisabled:            effBoolPtr(true),
	}, got.MetricsSources.SpanMetrics)

	assert.Equal(t, &model.MetricsSourceHostMetricsConfig{Disabled: effBoolPtr(true), Interval: effStrPtr("31s")},
		got.MetricsSources.HostMetrics)
	assert.Equal(t, &model.MetricsSourceKubeletStatsConfig{Disabled: effBoolPtr(true), Interval: effStrPtr("32s")},
		got.MetricsSources.KubeletStats)
	assert.Equal(t, &model.MetricsSourceOdigosOwnMetricsConfig{Interval: effStrPtr("33s")},
		got.MetricsSources.OdigosOwnMetrics)

	require.NotNil(t, got.MetricsSources.AgentMetrics)
	assert.Equal(t, &model.MetricsSourceAgentSpanMetricsConfig{Enabled: effBoolPtr(true)},
		got.MetricsSources.AgentMetrics.SpanMetrics)
	require.NotNil(t, got.MetricsSources.AgentMetrics.RuntimeMetrics)
	assert.Equal(t, &model.MetricsSourceAgentJavaRuntimeMetricsConfig{
		Disabled: effBoolPtr(true),
		Metrics: []*model.MetricsSourceAgentRuntimeMetricConfig{
			{Name: effStrPtr("jvm.memory.used"), Disabled: effBoolPtr(true)},
			{Name: effStrPtr("jvm.gc.duration")},
		},
	}, got.MetricsSources.AgentMetrics.RuntimeMetrics.Java)
}

func TestEffectiveConfigToModel_SurfacesTheSamplingConfig(t *testing.T) {
	got, err := EffectiveConfigToModel(fullyPopulatedOdigosConfig(), nil)
	require.NoError(t, err)

	assert.Equal(t, &model.SamplingConfig{
		DryRun: effBoolPtr(true),
		SpanSamplingAttributes: &model.SpanSamplingAttributesConfig{
			Disabled:                       effBoolPtr(true),
			SamplingCategoryDisabled:       effBoolPtr(true),
			TraceDecidingRuleDisabled:      effBoolPtr(true),
			SpanDecisionAttributesDisabled: effBoolPtr(true),
		},
		TailSampling: &model.TailSamplingConfig{
			Disabled:                     effBoolPtr(true),
			TraceAggregationWaitDuration: effStrPtr("11s"),
		},
		K8sHealthProbesSampling: &model.K8sHealthProbesSamplingConfig{
			Enabled:        effBoolPtr(true),
			KeepPercentage: effFloatPtr(12.5),
		},
	}, got.Sampling)
}

// Log levels are resolved, not copied: an unset component falls back to the configured
// default and, failing that, to info. The settings screen shows the level the component
// will actually run at, so the fallbacks matter more than the stored values.
func TestEffectiveConfigToModel_ResolvesComponentLogLevels(t *testing.T) {
	tests := []struct {
		name   string
		levels *common.ComponentLogLevels
		want   *model.ComponentLogLevelsConfig
	}{
		{
			name:   "no block at all falls back to info everywhere",
			levels: nil,
			want: &model.ComponentLogLevelsConfig{
				Default:      logLevelPtr(model.OdigosLogLevelInfo),
				Autoscaler:   logLevelPtr(model.OdigosLogLevelInfo),
				Scheduler:    logLevelPtr(model.OdigosLogLevelInfo),
				Instrumentor: logLevelPtr(model.OdigosLogLevelInfo),
				Odiglet:      logLevelPtr(model.OdigosLogLevelInfo),
				Deviceplugin: logLevelPtr(model.OdigosLogLevelInfo),
				UI:           logLevelPtr(model.OdigosLogLevelInfo),
				Collector:    logLevelPtr(model.OdigosLogLevelInfo),
			},
		},
		{
			name:   "a default with no overrides applies to every component",
			levels: &common.ComponentLogLevels{Default: common.LogLevelWarn},
			want: &model.ComponentLogLevelsConfig{
				Default:      logLevelPtr(model.OdigosLogLevelWarn),
				Autoscaler:   logLevelPtr(model.OdigosLogLevelWarn),
				Scheduler:    logLevelPtr(model.OdigosLogLevelWarn),
				Instrumentor: logLevelPtr(model.OdigosLogLevelWarn),
				Odiglet:      logLevelPtr(model.OdigosLogLevelWarn),
				Deviceplugin: logLevelPtr(model.OdigosLogLevelWarn),
				UI:           logLevelPtr(model.OdigosLogLevelWarn),
				Collector:    logLevelPtr(model.OdigosLogLevelWarn),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EffectiveConfigToModel(&common.OdigosConfiguration{ComponentLogLevels: tt.levels}, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got.ComponentLogLevels)
		})
	}
}

// There are eight components and only four log levels, so a single fixture cannot tell
// two components apart by value. Overriding exactly one at a time is what proves each
// model field is wired to its own component name rather than a neighbour's.
func TestEffectiveConfigToModel_EachComponentLogLevelReadsItsOwnComponent(t *testing.T) {
	tests := []struct {
		name     string
		levels   *common.ComponentLogLevels
		selector func(*model.ComponentLogLevelsConfig) *model.OdigosLogLevel
	}{
		{"autoscaler", &common.ComponentLogLevels{Autoscaler: common.LogLevelDebug}, func(c *model.ComponentLogLevelsConfig) *model.OdigosLogLevel { return c.Autoscaler }},
		{"scheduler", &common.ComponentLogLevels{Scheduler: common.LogLevelDebug}, func(c *model.ComponentLogLevelsConfig) *model.OdigosLogLevel { return c.Scheduler }},
		{"instrumentor", &common.ComponentLogLevels{Instrumentor: common.LogLevelDebug}, func(c *model.ComponentLogLevelsConfig) *model.OdigosLogLevel { return c.Instrumentor }},
		{"odiglet", &common.ComponentLogLevels{Odiglet: common.LogLevelDebug}, func(c *model.ComponentLogLevelsConfig) *model.OdigosLogLevel { return c.Odiglet }},
		{"deviceplugin", &common.ComponentLogLevels{Deviceplugin: common.LogLevelDebug}, func(c *model.ComponentLogLevelsConfig) *model.OdigosLogLevel { return c.Deviceplugin }},
		{"ui", &common.ComponentLogLevels{UI: common.LogLevelDebug}, func(c *model.ComponentLogLevelsConfig) *model.OdigosLogLevel { return c.UI }},
		{"collector", &common.ComponentLogLevels{Collector: common.LogLevelDebug}, func(c *model.ComponentLogLevelsConfig) *model.OdigosLogLevel { return c.Collector }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EffectiveConfigToModel(&common.OdigosConfiguration{ComponentLogLevels: tt.levels}, nil)
			require.NoError(t, err)

			levels := got.ComponentLogLevels
			require.NotNil(t, levels)
			assert.Equal(t, logLevelPtr(model.OdigosLogLevelDebug), tt.selector(levels))

			// Every other component keeps the info fallback, so an override cannot leak.
			overridden := 0
			for _, level := range []*model.OdigosLogLevel{
				levels.Default, levels.Autoscaler, levels.Scheduler, levels.Instrumentor,
				levels.Odiglet, levels.Deviceplugin, levels.UI, levels.Collector,
			} {
				require.NotNil(t, level)
				if *level == model.OdigosLogLevelDebug {
					overridden++
				}
			}
			assert.Equal(t, 1, overridden)
		})
	}
}

func logLevelPtr(v model.OdigosLogLevel) *model.OdigosLogLevel { return &v }

// An empty effective config is what a UI talking to a partially reconciled cluster sees.
// The wrapper objects the schema declares must still be present (the UI dereferences
// them), while every optional scalar must stay absent rather than render as false or "".
func TestEffectiveConfigToModel_EmptyConfigLeavesOptionalFieldsAbsent(t *testing.T) {
	got, err := EffectiveConfigToModel(&common.OdigosConfiguration{}, nil)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.NotNil(t, got.AllowConcurrentAgents)
	assert.Nil(t, got.AllowConcurrentAgents.Enabled)
	assert.NotNil(t, got.Karpenter)
	assert.Nil(t, got.Karpenter.Enabled)
	assert.NotNil(t, got.Instrumentor)
	assert.Nil(t, got.Instrumentor.MountMethod)
	assert.Nil(t, got.Instrumentor.AgentEnvVarsInjectionMethod)
	assert.Nil(t, got.Instrumentor.CheckDeviceHealthBeforeInjection)
	assert.NotNil(t, got.AutoRollback)
	assert.Nil(t, got.AutoRollback.Disabled)
	assert.NotNil(t, got.ComponentLogLevels)

	assert.Nil(t, got.IgnoreOdigosNamespace)
	assert.Nil(t, got.ClickhouseJSONTypeEnabled)
	assert.Nil(t, got.ImagePrefix)
	assert.Nil(t, got.ClusterName)
	assert.Nil(t, got.UIMode)
	assert.Nil(t, got.UIPaginationLimit)
	assert.Nil(t, got.OdigletHealthProbeBindPort)
	assert.Nil(t, got.NodeSelector)
	assert.Nil(t, got.CollectorGateway)
	assert.Nil(t, got.CollectorNode)
	assert.Nil(t, got.Rollout)
	assert.Nil(t, got.Oidc)
	assert.Nil(t, got.UserInstrumentationEnvs)
	assert.Nil(t, got.MetricsSources)
	assert.Nil(t, got.AgentsInitContainerResources)
	assert.Nil(t, got.OdigosOwnTelemetryStore)
	assert.Nil(t, got.Sampling)
	assert.Nil(t, got.Profiling)
	assert.Nil(t, got.TraceCorrelations)
	assert.Empty(t, got.Profiles)

	// The four non-pointer booleans are always reported, so the UI can tell "false" from
	// "not configured" for everything else.
	assert.Equal(t, effBoolPtr(false), got.TelemetryEnabled)
	assert.Equal(t, effBoolPtr(false), got.OpenshiftEnabled)
	assert.Equal(t, effBoolPtr(false), got.Psp)
	assert.Equal(t, effBoolPtr(false), got.SkipWebhookIssuerCreation)

	assert.Equal(t, alwaysRecordedProvenancePaths, recordedProvenancePaths(got.Provenance))
}

// The four always-reported booleans all render as true in the fully populated fixture,
// which cannot tell them apart. Setting one at a time is what proves each model field
// reads its own configuration field.
func TestEffectiveConfigToModel_EachAlwaysReportedBooleanReadsItsOwnField(t *testing.T) {
	tests := []struct {
		name     string
		config   *common.OdigosConfiguration
		selector func(*model.EffectiveConfig) *bool
	}{
		{"telemetryEnabled", &common.OdigosConfiguration{TelemetryEnabled: true}, func(c *model.EffectiveConfig) *bool { return c.TelemetryEnabled }},
		{"openshiftEnabled", &common.OdigosConfiguration{OpenshiftEnabled: true}, func(c *model.EffectiveConfig) *bool { return c.OpenshiftEnabled }},
		{"psp", &common.OdigosConfiguration{Psp: true}, func(c *model.EffectiveConfig) *bool { return c.Psp }},
		{"skipWebhookIssuerCreation", &common.OdigosConfiguration{SkipWebhookIssuerCreation: true}, func(c *model.EffectiveConfig) *bool { return c.SkipWebhookIssuerCreation }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EffectiveConfigToModel(tt.config, nil)
			require.NoError(t, err)

			assert.Equal(t, effBoolPtr(true), tt.selector(got))

			enabled := 0
			for _, v := range []*bool{got.TelemetryEnabled, got.OpenshiftEnabled, got.Psp, got.SkipWebhookIssuerCreation} {
				require.NotNil(t, v)
				if *v {
					enabled++
				}
			}
			assert.Equal(t, 1, enabled)
		})
	}
}

// A nested block present but empty must not invent values for the fields inside it.
func TestEffectiveConfigToModel_EmptyNestedBlocksSurfaceNoValues(t *testing.T) {
	got, err := EffectiveConfigToModel(&common.OdigosConfiguration{
		CollectorGateway:  &common.CollectorGatewayConfiguration{},
		CollectorNode:     &common.CollectorNodeConfiguration{},
		Oidc:              &common.OidcConfiguration{},
		MetricsSources:    &common.MetricsSourceConfiguration{},
		Sampling:          &common.SamplingConfiguration{},
		Profiling:         &common.ProfilingConfiguration{},
		TraceCorrelations: &common.TraceCorrelationsConfiguration{},
		Rollout:           &common.RolloutConfiguration{},
	}, nil)
	require.NoError(t, err)

	assert.Equal(t, &model.CollectorGatewayConfig{}, got.CollectorGateway)
	assert.Equal(t, &model.CollectorNodeConfig{}, got.CollectorNode)
	assert.Equal(t, &model.OidcConfig{}, got.Oidc)
	assert.Equal(t, &model.MetricsSourceConfig{}, got.MetricsSources)
	assert.Equal(t, &model.SamplingConfig{}, got.Sampling)
	assert.Equal(t, &model.ProfilingConfig{}, got.Profiling)
	assert.Equal(t, &model.TraceCorrelationsConfig{ServiceIo: &model.TraceCorrelationsServiceIOConfig{}}, got.TraceCorrelations)
	assert.Equal(t, &model.RolloutConfig{}, got.Rollout)

	// An empty rollout block still reports automaticRolloutDisabled, because the value the
	// UI renders for it (nil, meaning "rollouts are automatic") is a real answer.
	assert.Equal(t,
		append(append([]string{}, alwaysRecordedProvenancePaths...), "rollout.automaticRolloutDisabled"),
		recordedProvenancePaths(got.Provenance))
}

// alwaysRecordedProvenancePaths are reported even for a completely empty configuration,
// because their rendered value ("false", "no namespaces ignored", "info") is meaningful
// rather than absent. In recorded order.
var alwaysRecordedProvenancePaths = []string{
	"telemetryEnabled",
	"openshiftEnabled",
	"psp",
	"skipWebhookIssuerCreation",
	"ignoredNamespaces",
	"ignoredContainers",
	"allowedTestConnectionHosts",
	"imagePullSecrets",
	"profiles",
	"componentLogLevels.default",
	"componentLogLevels.autoscaler",
	"componentLogLevels.scheduler",
	"componentLogLevels.instrumentor",
	"componentLogLevels.odiglet",
	"componentLogLevels.deviceplugin",
	"componentLogLevels.ui",
	"componentLogLevels.collector",
}

func recordedProvenancePaths(entries []*model.ProvenanceEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.HelmPath)
	}
	return paths
}

// The gateway node selector is the one field on this path that can fail to serialise;
// everything else is copied. Both node selectors round trip as compact JSON because the
// UI parses them back.
func TestEffectiveConfigToModel_NodeSelectorsAreCompactJSON(t *testing.T) {
	gatewayNodeSelector := map[string]string{"b": "2", "a": "1"}

	got, err := EffectiveConfigToModel(&common.OdigosConfiguration{
		NodeSelector:     map[string]string{"kubernetes.io/os": "linux"},
		CollectorGateway: &common.CollectorGatewayConfiguration{NodeSelector: &gatewayNodeSelector},
	}, nil)
	require.NoError(t, err)

	require.NotNil(t, got.NodeSelector)
	var roundTripped map[string]string
	require.NoError(t, json.Unmarshal([]byte(*got.NodeSelector), &roundTripped))
	assert.Equal(t, map[string]string{"kubernetes.io/os": "linux"}, roundTripped)

	require.NotNil(t, got.CollectorGateway.NodeSelector)
	assert.Equal(t, `{"a":"1","b":"2"}`, *got.CollectorGateway.NodeSelector)
}

// An empty gateway node selector map is not the same as a configured one, and must not
// render as the literal "{}" in a text box.
func TestEffectiveConfigToModel_EmptyGatewayNodeSelectorIsAbsent(t *testing.T) {
	empty := map[string]string{}

	got, err := EffectiveConfigToModel(&common.OdigosConfiguration{
		CollectorGateway: &common.CollectorGatewayConfiguration{NodeSelector: &empty},
	}, nil)

	require.NoError(t, err)
	assert.Nil(t, got.CollectorGateway.NodeSelector)
}

// The per-block converters each guard their own nil input even though the caller already
// checks. Keeping the guards honest is what lets a future caller reuse them (the sampling
// converter is already called from two places).
func TestEffectiveConfigConvertersToleratesNilInput(t *testing.T) {
	pc := newProvenanceCollector(nil)

	gateway, err := convertCollectorGatewayToModel(nil, pc)
	require.NoError(t, err)
	assert.Nil(t, gateway)

	envs, err := convertUserInstrumentationEnvsToModel(nil, pc)
	require.NoError(t, err)
	assert.Nil(t, envs)

	assert.Nil(t, convertCollectorNodeToModel(nil, pc))
	assert.Nil(t, convertOtlpExporterToModel(nil, pc))
	assert.Nil(t, convertMetricsSourcesToModel(nil, pc))
	assert.Nil(t, convertOdigosConfigToSamplingConfig(nil))
	assert.Nil(t, convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{}))

	recordSamplingProvenance(nil, pc)
	assert.Empty(t, pc.entries)
}

// A language map that carries no languages is not a configured value.
func TestEffectiveConfigToModel_EmptyUserInstrumentationEnvsSurfacesNoLanguages(t *testing.T) {
	got, err := EffectiveConfigToModel(&common.OdigosConfiguration{
		UserInstrumentationEnvs: &common.UserInstrumentationEnvs{},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, &model.UserInstrumentationEnvsConfig{}, got.UserInstrumentationEnvs)
}

func TestConvertUiModeToModel(t *testing.T) {
	assert.Equal(t, model.UIModeReadonly, convertUiModeToModel(common.UiModeReadonly))
	assert.Equal(t, model.UIModeDefault, convertUiModeToModel(common.UiModeDefault))
	assert.Equal(t, model.UIModeDefault, convertUiModeToModel(common.UiMode("something-else")))
}

func TestRemoteConfigToModel(t *testing.T) {
	assert.Nil(t, RemoteConfigToModel(nil))
	assert.Equal(t, &model.RemoteConfig{}, RemoteConfigToModel(&common.OdigosConfiguration{}))
	assert.Equal(t,
		&model.RemoteConfig{Rollout: &model.RemoteConfigRollout{AutomaticRolloutDisabled: effBoolPtr(true)}},
		RemoteConfigToModel(&common.OdigosConfiguration{
			Rollout: &common.RolloutConfiguration{AutomaticRolloutDisabled: effBoolPtr(true)},
		}))
}

func provenanceByHelmPath(entries []*model.ProvenanceEntry) map[string]string {
	byPath := make(map[string]string, len(entries))
	for _, e := range entries {
		byPath[e.HelmPath] = e.ReconciledFrom
	}
	return byPath
}

// Every value the model surfaces gets a provenance entry, and a field the provenance map
// says nothing about is attributed to the helm baseline rather than left blank.
func TestEffectiveConfigToModel_UnattributedFieldsFallBackToTheHelmBaseline(t *testing.T) {
	got, err := EffectiveConfigToModel(fullyPopulatedOdigosConfig(), nil)
	require.NoError(t, err)

	require.NotEmpty(t, got.Provenance)
	for _, entry := range got.Provenance {
		assert.Equal(t, "odigos-configuration", entry.ReconciledFrom, "helm path %q", entry.HelmPath)
	}
}

// The provenance map is keyed by the YAML config key while the UI renders the helm value
// path, and the two differ for several fields. recordAs is what bridges them, so a field
// that switches to the plain recorder silently loses its overlay badge.
func TestEffectiveConfigToModel_HelmPathsAndYamlKeysAreBridged(t *testing.T) {
	prov := map[string]string{
		"allowConcurrentAgents":            consts.OdigosLocalUiConfigName,
		"karpenterEnabled":                 consts.OdigosRemoteConfigName,
		"checkDeviceHealthBeforeInjection": consts.OdigosLocalUiConfigName,
		"mountMethod":                      "profile",
		"agentEnvVarsInjectionMethod":      consts.OdigosRemoteConfigName,
		"rollbackDisabled":                 consts.OdigosLocalUiConfigName,
		"rollbackGraceTime":                consts.OdigosLocalUiConfigName,
		"rollbackStabilityWindow":          consts.OdigosLocalUiConfigName,
	}

	got, err := EffectiveConfigToModel(fullyPopulatedOdigosConfig(), prov)
	require.NoError(t, err)

	byPath := provenanceByHelmPath(got.Provenance)
	assert.Equal(t, map[string]string{
		"allowConcurrentAgents.enabled":                 consts.OdigosLocalUiConfigName,
		"karpenter.enabled":                             consts.OdigosRemoteConfigName,
		"instrumentor.checkDeviceHealthBeforeInjection": consts.OdigosLocalUiConfigName,
		"instrumentor.mountMethod":                      "profile",
		"instrumentor.agentEnvVarsInjectionMethod":      consts.OdigosRemoteConfigName,
		"autoRollback.disabled":                         consts.OdigosLocalUiConfigName,
		"autoRollback.graceTime":                        consts.OdigosLocalUiConfigName,
		"autoRollback.stabilityWindowTime":              consts.OdigosLocalUiConfigName,
	}, map[string]string{
		"allowConcurrentAgents.enabled":                 byPath["allowConcurrentAgents.enabled"],
		"karpenter.enabled":                             byPath["karpenter.enabled"],
		"instrumentor.checkDeviceHealthBeforeInjection": byPath["instrumentor.checkDeviceHealthBeforeInjection"],
		"instrumentor.mountMethod":                      byPath["instrumentor.mountMethod"],
		"instrumentor.agentEnvVarsInjectionMethod":      byPath["instrumentor.agentEnvVarsInjectionMethod"],
		"autoRollback.disabled":                         byPath["autoRollback.disabled"],
		"autoRollback.graceTime":                        byPath["autoRollback.graceTime"],
		"autoRollback.stabilityWindowTime":              byPath["autoRollback.stabilityWindowTime"],
	})

	// The bridged fields must not also be reported under their YAML key, or the UI would
	// render the same setting twice with two different badges.
	for _, yamlKey := range []string{"allowConcurrentAgents", "karpenterEnabled", "checkDeviceHealthBeforeInjection",
		"mountMethod", "agentEnvVarsInjectionMethod", "rollbackDisabled", "rollbackGraceTime", "rollbackStabilityWindow"} {
		assert.NotContains(t, byPath, yamlKey)
	}
}

// A path recorded twice would render two rows for one setting in the UI.
func TestEffectiveConfigToModel_ProvenancePathsAreUnique(t *testing.T) {
	got, err := EffectiveConfigToModel(fullyPopulatedOdigosConfig(), nil)
	require.NoError(t, err)

	seen := map[string]int{}
	for _, entry := range got.Provenance {
		seen[entry.HelmPath]++
	}
	for path, count := range seen {
		assert.Equal(t, 1, count, "helm path %q recorded %d times", path, count)
	}
}

// Absent values must not be attributed to anything: a provenance entry for a field the
// model did not surface is a row the UI cannot render.
func TestEffectiveConfigToModel_AbsentValuesAreNotAttributed(t *testing.T) {
	got, err := EffectiveConfigToModel(&common.OdigosConfiguration{}, nil)
	require.NoError(t, err)

	byPath := provenanceByHelmPath(got.Provenance)
	for _, path := range []string{
		"clusterName", "imagePrefix", "uiMode", "instrumentor.mountMethod",
		"instrumentor.agentEnvVarsInjectionMethod", "allowConcurrentAgents.enabled",
		"karpenter.enabled", "autoRollback.disabled", "oidc.tenantUrl",
		"collectorGateway.minReplicas", "collectorNode.requestCPUm",
		"sampling.dryRun", "profiling.enabled", "nodeSelector",
	} {
		assert.NotContains(t, byPath, path)
	}
}

// odigosConfigFieldsSurfacedInTheUI lists every common.OdigosConfiguration field that the
// settings screen is expected to render. odigosConfigFieldsNotSurfaced lists the rest,
// with the reason. Together they must account for the whole struct, so adding a field to
// the odigos configuration forces an explicit decision about whether the UI shows it -
// which is exactly the step that gets skipped.
var odigosConfigFieldsSurfacedInTheUI = []string{
	"agentEnvVarsInjectionMethod",
	"agentsInitContainerResources",
	"allowConcurrentAgents",
	"allowedTestConnectionHosts",
	"centralBackendURL",
	"checkDeviceHealthBeforeInjection",
	"clickhouseJsonTypeEnabled",
	"clusterName",
	"collectorGateway",
	"collectorNode",
	"componentLogLevels",
	"configVersion",
	"customContainerRuntimeSocketPath",
	"goAutoOffsetsCron",
	"goAutoOffsetsMode",
	"ignoreOdigosNamespace",
	"ignoredContainers",
	"ignoredNamespaces",
	"imagePrefix",
	"imagePullSecrets",
	"karpenterEnabled",
	"metricsSources",
	"mountMethod",
	"nodeSelector",
	"odigletHealthProbeBindPort",
	"odigosOwnTelemetryStore",
	"oidc",
	"openshiftEnabled",
	"profiles",
	"profiling",
	"psp",
	"resourceSizePreset",
	"rollbackDisabled",
	"rollbackGraceTime",
	"rollbackStabilityWindow",
	"rollout",
	"sampling",
	"skipWebhookIssuerCreation",
	"telemetryEnabled",
	"traceCorrelations",
	"traceIdSuffix",
	"uiMode",
	"uiPaginationLimit",
	"uiRemoteUrl",
	"userInstrumentationEnvs",
}

var odigosConfigFieldsNotSurfaced = map[string]string{
	"insights":      "insights has its own settings surface, served by frontend/services/insights",
	"mcpAccessMode": "the MCP access mode is written through SetMcpAccessMode and read by the MCP server, not the settings screen",
	"mcpEnabled":    "the MCP toggle is served by the local UI config resolver, not the effective config",
}

func TestEveryOdigosConfigurationFieldIsClassifiedForTheSettingsScreen(t *testing.T) {
	configType := reflect.TypeOf(common.OdigosConfiguration{})

	actual := make([]string, 0, configType.NumField())
	for i := 0; i < configType.NumField(); i++ {
		jsonTag := configType.Field(i).Tag.Get("json")
		require.NotEmpty(t, jsonTag, "field %q has no json tag", configType.Field(i).Name)

		name, _, _ := strings.Cut(jsonTag, ",")
		actual = append(actual, name)
	}

	classified := append([]string{}, odigosConfigFieldsSurfacedInTheUI...)
	for name := range odigosConfigFieldsNotSurfaced {
		classified = append(classified, name)
	}

	sort.Strings(actual)
	sort.Strings(classified)
	assert.Equal(t, actual, classified,
		"a field was added to or removed from common.OdigosConfiguration; decide whether the settings "+
			"screen surfaces it and update EffectiveConfigToModel plus one of the two lists here")
}

// effectiveConfigFieldsNotPopulated are the model fields EffectiveConfigToModel leaves
// alone. Everything else must come out non-zero for a fully populated configuration,
// which is what catches a conversion that quietly stops assigning a field.
var effectiveConfigFieldsNotPopulated = map[string]string{
	"ManifestYaml": "filled in by the resolver from the raw ConfigMap, not by this conversion",
	"Wasp": "left over from the wasp removal in #5745: the GraphQL schema still declares " +
		"EffectiveConfig.wasp and the webapp still selects it, but nothing populates it any more",
}

func TestEffectiveConfigToModel_PopulatesEveryModelField(t *testing.T) {
	got, err := EffectiveConfigToModel(fullyPopulatedOdigosConfig(), nil)
	require.NoError(t, err)

	value := reflect.ValueOf(*got)
	for i := 0; i < value.NumField(); i++ {
		name := value.Type().Field(i).Name
		if _, expected := effectiveConfigFieldsNotPopulated[name]; expected {
			assert.True(t, value.Field(i).IsZero(),
				"model.EffectiveConfig.%s is populated now; drop it from effectiveConfigFieldsNotPopulated", name)
			continue
		}
		assert.False(t, value.Field(i).IsZero(),
			"model.EffectiveConfig.%s is not populated from a fully populated odigos configuration", name)
	}
}
