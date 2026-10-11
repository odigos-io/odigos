package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/odigos-io/odigos/instrumentor/controllers/collectorhooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestAddSelfTelemetryPipeline(t *testing.T) {
	var empty = struct{}{}
	cases := []struct {
		name string
		cfg  *config.Config
		err  error
	}{
		{
			name: "no pipeline",
			cfg: &config.Config{
				Exporters: map[string]interface{}{},
				Receivers: map[string]interface{}{},
			},
			err: errNoPipelineConfigured,
		},
		{
			name: "no receivers",
			cfg: &config.Config{
				Exporters: map[string]interface{}{},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{},
				},
			},
			err: errNoReceiversConfigured,
		},
		{
			name: "no exporters",
			cfg: &config.Config{
				Receivers: map[string]interface{}{},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{},
				},
			},
			err: errNoExportersConfigured,
		},
		{
			name: "minimal config",
			cfg: &config.Config{
				Receivers: map[string]interface{}{},
				Exporters: map[string]interface{}{},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{},
				},
			},
			err: nil,
		},
		{
			name: "config with pipeline",
			cfg: &config.Config{
				Receivers: config.GenericMap{
					"otlp": config.GenericMap{
						"protocols": config.GenericMap{
							"grpc": empty,
							"http": empty,
						},
					},
				},
				Processors: config.GenericMap{
					"memory_limiter": config.GenericMap{
						"check_interval": "500ms",
					},
					"resource/odigos-version": config.GenericMap{
						"attributes": []config.GenericMap{
							{
								"key":    "odigos.version",
								"value":  "${ODIGOS_VERSION}",
								"action": "upsert",
							},
						},
					},
				},
				Exporters: config.GenericMap{
					"otlp_grpc/local": config.GenericMap{
						"endpoint": "http://localhost:4317",
						"tls": config.GenericMap{
							"insecure": true,
						},
					},
				},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{
						"traces/user-pipeline": {
							Receivers:  []string{"otlp"},
							Processors: []string{"memory_limiter", "resource/odigos-version"},
							Exporters:  []string{"otlp_grpc/local"},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.cfg
			err := addSelfTelemetryPipeline(c, k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault, []string{"traces/user-pipeline"}, []string{})
			if !assert.ErrorIs(t, err, tc.err) {
				return
			}
			if err != nil {
				return
			}
			assert.NotEmpty(t, c.Receivers["prometheus/self-metrics"])
			assert.NotEmpty(t, c.Processors["resource/pod-name"])
			assert.NotEmpty(t, c.Processors["resource/odigos-collector-role"])
			assert.NotEmpty(t, c.Service.Pipelines["metrics/otelcol"])
			assert.Equal(t, []string{"prometheus/self-metrics"}, c.Service.Pipelines["metrics/otelcol"].Receivers)
			assert.Equal(t, []string{"resource/pod-name", "resource/odigos-collector-role"}, c.Service.Pipelines["metrics/otelcol"].Processors)
			assert.Equal(t, []string{"otlp_grpc/odigos-own-telemetry-ui"}, c.Service.Pipelines["metrics/otelcol"].Exporters)
			pullExporter := c.Service.Telemetry.Metrics.Readers[0]["pull"].(config.GenericMap)["exporter"].(config.GenericMap)
			prometheusExporter := pullExporter["prometheus"].(config.GenericMap)
			port := prometheusExporter["port"]
			address := prometheusExporter["host"]
			assert.Equal(t, k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault, port)
			assert.Equal(t, "0.0.0.0", address)
			assert.Equal(t, false, prometheusExporter["without_units"])
			assert.Equal(t, false, prometheusExporter["without_type_suffix"])
			for pipelineName, pipeline := range c.Service.Pipelines {
				if pipelineName == "metrics/otelcol" {
					assert.NotContains(t, pipeline.Processors, "odigostrafficmetrics")
				} else {
					assert.Equal(t, pipeline.Processors[len(pipeline.Processors)-1], "odigostrafficmetrics")
				}
			}
		})
	}
}

func TestAddSelfTelemetryPipeline_ProfilesPipelineIncludesTrafficMetrics(t *testing.T) {
	// the odigostrafficmetrics processor supports the profiles signal,
	// so it should be appended to profiles destination pipelines like any other signal.
	c := &config.Config{
		Receivers: config.GenericMap{"otlp": struct{}{}},
		Exporters: config.GenericMap{"otlp_grpc/local": config.GenericMap{}},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				"profiles/generic-dest": {
					Receivers: []string{"otlp"},
					Exporters: []string{"otlp_grpc/local"},
				},
			},
		},
	}

	err := addSelfTelemetryPipeline(c, k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault, []string{"profiles/generic-dest"}, []string{})
	assert.NoError(t, err)
	assert.Contains(t, c.Service.Pipelines["profiles/generic-dest"].Processors, "odigostrafficmetrics")
}

const hookExporterName = "test/hook-exporter"

type clusterConfigHook struct{}

func (clusterConfigHook) MutateConfig(in collectorhooks.GatewayHookInput) error {
	in.AddExporter(hookExporterName, config.GenericMap{"endpoint": "x"})
	in.AppendExporterToRootTraces(hookExporterName)
	return nil
}

func (clusterConfigHook) MutateDeployment(in *collectorhooks.GatewayDeploymentHookInput) {
	in.AddEnv(corev1.EnvVar{Name: "TEST_HOOK_ENV", Value: "1"})
	in.AddFeatureGate("test.hookGate")
}

type debugMetricsDest struct{ id string }

func (d debugMetricsDest) GetID() string                   { return d.id }
func (d debugMetricsDest) GetType() common.DestinationType { return common.DebugDestinationType }
func (d debugMetricsDest) GetConfig() map[string]string    { return map[string]string{} }
func (d debugMetricsDest) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.MetricsObservabilitySignal}
}

func gatewayConfigWithHook(t *testing.T, dests []config.ExporterConfigurer, hooks *collectorhooks.CollectorHooks) *config.Config {
	t.Helper()
	ext := k8sconsts.OdigosConfigK8sExtensionType
	off := false
	opts := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:           "odigos-system",
		OdigosConfigExtensionName: &ext,
		TailSamplingEnabled:       &off,
	}
	cfg, err, _, _ := pipelinegen.CalculateGatewayConfig(dests, nil,
		func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error {
			if err := addSelfTelemetryPipeline(c, 8888, destinationPipelineNames, signalsRootPipelines); err != nil {
				return err
			}
			in := collectorhooks.GatewayHookInput{OdigosNamespace: "odigos-system"}
			in.Config = c
			return collectorhooks.ApplyGatewayConfig(hooks, in)
		}, nil, &opts)
	require.NoError(t, err)
	return cfg
}

func anyPipelineHasExporter(cfg *config.Config, name string) bool {
	for _, p := range cfg.Service.Pipelines {
		for _, exp := range p.Exporters {
			if exp == name {
				return true
			}
		}
	}
	return false
}

func TestClusterCollectorHook_NilIsCommunity(t *testing.T) {
	cfg := gatewayConfigWithHook(t, []config.ExporterConfigurer{healthyJaegerDest{id: "j1"}}, nil)
	assert.NotContains(t, cfg.Exporters, hookExporterName)
	assert.False(t, anyPipelineHasExporter(cfg, hookExporterName))
}

func TestClusterCollectorHook_AppendsExporterWhenTracesExist(t *testing.T) {
	hooks := &collectorhooks.CollectorHooks{ClusterCollectorConfig: clusterConfigHook{}}
	cfg := gatewayConfigWithHook(t, []config.ExporterConfigurer{healthyJaegerDest{id: "j1"}}, hooks)
	assert.Contains(t, cfg.Exporters, hookExporterName)
	assert.True(t, anyPipelineHasExporter(cfg, hookExporterName),
		"hook exporter should be appended on the community root traces pipeline")
}

func TestClusterCollectorHook_MissingTracesPipelineIsNoop(t *testing.T) {
	hooks := &collectorhooks.CollectorHooks{ClusterCollectorConfig: clusterConfigHook{}}
	cfg := gatewayConfigWithHook(t, []config.ExporterConfigurer{debugMetricsDest{id: "m1"}}, hooks)
	assert.Contains(t, cfg.Exporters, hookExporterName)
	assert.False(t, anyPipelineHasExporter(cfg, hookExporterName),
		"AppendExporterToRootTraces must not create a traces pipeline")
}

func TestClusterCollectorHook_MutateDeployment(t *testing.T) {
	dep := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "col", Env: []corev1.EnvVar{{Name: "POD_NAME"}}}},
				},
			},
		},
	}
	in := &collectorhooks.GatewayDeploymentHookInput{Deployment: dep}
	collectorhooks.ApplyGatewayDeployment(&collectorhooks.CollectorHooks{ClusterCollectorConfig: clusterConfigHook{}}, in)
	require.Len(t, dep.Spec.Template.Spec.Containers[0].Env, 2)
	assert.Equal(t, "TEST_HOOK_ENV", dep.Spec.Template.Spec.Containers[0].Env[1].Name)
	assert.Equal(t, []string{"test.hookGate"}, in.FeatureGates())
}
