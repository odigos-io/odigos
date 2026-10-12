package clustercollector

import (
	"slices"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/odigos-io/odigos/common/urltemplate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// utLearning nests a live traffic learning block at the full
// cardinalityControl.urlTemplatization.liveTrafficLearning depth the
// OdigosConfiguration uses.
func utLearning(ltl *common.LiveTrafficLearningConfiguration) *common.CardinalityControlConfiguration {
	return &common.CardinalityControlConfiguration{
		UrlTemplatization: &common.UrlTemplatizationCardinalityControlConfiguration{
			LiveTrafficLearning: ltl,
		},
	}
}

// utEnabledLearning returns a fully nested config with the opt-in flag explicitly on.
func utEnabledLearning(ltl *common.LiveTrafficLearningConfiguration) *common.CardinalityControlConfiguration {
	on := true
	ltl.Enabled = &on
	return utLearning(ltl)
}

func utIntPtr(v int) *int { return &v }

// utDisabledShapes enumerates every way the unmatched-path side channel must stay
// off. The exporter is opt-in at four nesting levels, so each level needs its own
// fixture: a guard dropped from the chain is invisible to a fixture that already
// fails an earlier guard.
func utDisabledShapes() map[string]*common.CardinalityControlConfiguration {
	off := false
	return map[string]*common.CardinalityControlConfiguration{
		"nil cardinalityControl":  nil,
		"nil urlTemplatization":   {},
		"nil liveTrafficLearning": utLearning(nil),
		"liveTrafficLearning no enabled": utLearning(&common.LiveTrafficLearningConfiguration{
			// Every tunable set but the opt-in flag left unset: tuning the feature
			// must not be what turns it on.
			MaxExamplePathsPerWorkload: utIntPtr(10),
			PathExampleIdleTTL:         "1h",
			LearningInterval:           "15s",
		}),
		"liveTrafficLearning disabled": utLearning(&common.LiveTrafficLearningConfiguration{Enabled: &off}),
	}
}

// odigos-cache is only rendered by helm behind cacheDb.enabled, which in turn is
// gated on an on-prem token, and the odigos_url_templatization exporter itself
// ships in the enterprise collector build. Wiring it on community tier would make
// the OSS gateway fail to start on an unknown component type.
func TestEffectiveCardinalityControl(t *testing.T) {
	cfg := utEnabledLearning(&common.LiveTrafficLearningConfiguration{})

	assert.Nil(t, effectiveCardinalityControl(cfg, common.CommunityOdigosTier),
		"community tier must not wire an exporter that only exists in the enterprise collector")
	assert.Same(t, cfg, effectiveCardinalityControl(cfg, common.OnPremOdigosTier))
	assert.Same(t, cfg, effectiveCardinalityControl(cfg, common.CloudOdigosTier))
	assert.Nil(t, effectiveCardinalityControl(nil, common.OnPremOdigosTier))

	// OdigosTier.IsEnterprise is `!= community`, so an unset tier is treated as
	// entitled. syncConfigMap reads the tier from the odigos-deployment ConfigMap,
	// where a missing key yields "". Pinned so the fail-open direction is a
	// deliberate choice rather than an accident of the comparison.
	assert.Same(t, cfg, effectiveCardinalityControl(cfg, common.OdigosTier("")),
		"an unresolved tier currently falls through to the enterprise behaviour")
}

func TestAddUrlTemplatizationUnmatchedExporter_DisabledIsAZeroWriteNoop(t *testing.T) {
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)

	for name, cc := range utDisabledShapes() {
		t.Run(name, func(t *testing.T) {
			c := configWithTracesIn()
			require.NoError(t, addUrlTemplatizationUnmatchedExporter(c, "odigos-system", cc))

			assert.NotContains(t, c.Exporters, commonconf.UrlTemplatizationExporter,
				"exporter must not be registered while live traffic learning is off")
			// The root pipeline list must be byte-identical, not merely missing the
			// new name: a refusal that still rewrites traces/in would churn the
			// gateway ConfigMap and restart the collector on every reconcile.
			assert.Equal(t, []string{"odigosrouterconnector/traces"}, c.Service.Pipelines[rootName].Exporters)
		})
	}
}

func TestAddUrlTemplatizationUnmatchedExporter_NoRootTracesPipelineIsANoop(t *testing.T) {
	c := &config.Config{Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
	require.NoError(t, addUrlTemplatizationUnmatchedExporter(c, "odigos-system",
		utEnabledLearning(&common.LiveTrafficLearningConfiguration{})))

	assert.NotContains(t, c.Exporters, commonconf.UrlTemplatizationExporter,
		"there is no root traces pipeline to tap, so nothing must be registered")
	assert.Empty(t, c.Service.Pipelines, "no pipeline may be invented for the side channel")
}

func TestAddUrlTemplatizationUnmatchedExporter_EnabledAppendsExporterToRootPipeline(t *testing.T) {
	c := configWithTracesIn()
	require.NoError(t, addUrlTemplatizationUnmatchedExporter(c, "odigos-system",
		utEnabledLearning(&common.LiveTrafficLearningConfiguration{})))

	exp, ok := c.Exporters[commonconf.UrlTemplatizationExporter].(config.GenericMap)
	require.True(t, ok, "exporter must be registered")

	// Whole-map equality rather than per-key assertions: the four settings are all
	// read by the same enterprise exporter, so a value written under the wrong key
	// is silently dropped and the exporter falls back to its own defaults.
	assert.Equal(t, config.GenericMap{
		"odigos_config_extension": "odigos_config_k8s",
		"endpoint":                "odigos-cache.odigos-system:6379",
		"max_paths_per_workload":  5000,
		"path_idle_ttl":           "48h",
	}, exp)

	rootPipe := c.Service.Pipelines[pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)]
	assert.Equal(t, []string{"resource/odigos-version", "transform/url-template"}, rootPipe.Processors,
		"the side channel must observe the root processors, so the chain is preserved verbatim")
	assert.Equal(t, []string{"odigosrouterconnector/traces", commonconf.UrlTemplatizationExporter}, rootPipe.Exporters,
		"the exporter fans out alongside the destination router, it does not replace it")
}

// The instance name doubles as the otelcol component TYPE (there is no "/name"
// suffix), and the factory that implements it lives in the odigos-enterprise
// collector build, not in this repository. Renaming it here makes the gateway
// refuse to start with "unknown type", so the literal is pinned rather than read
// back from the constant it came from.
func TestUrlTemplatizationExporterComponentName(t *testing.T) {
	assert.Equal(t, "odigos_url_templatization", commonconf.UrlTemplatizationExporter)
	assert.NotContains(t, commonconf.UrlTemplatizationExporter, "/",
		"the exporter is registered under its bare component type")
}

// The endpoint is assembled from two same-typed strings and must address the
// odigos-cache ClusterIP Service that helm/odigos/templates/cache/service.yaml
// renders (name odigos-cache, port 6379) in the Odigos release namespace. Swap the
// two and the gateway resolves a host that does not exist, so unmatched paths are
// dropped with a connection error per batch and no rule is ever learned.
func TestAddUrlTemplatizationUnmatchedExporter_EndpointAddressesOdigosCacheInTheOdigosNamespace(t *testing.T) {
	for _, ns := range []string{"odigos-system", "custom-odigos-ns"} {
		t.Run(ns, func(t *testing.T) {
			c := configWithTracesIn()
			require.NoError(t, addUrlTemplatizationUnmatchedExporter(c, ns,
				utEnabledLearning(&common.LiveTrafficLearningConfiguration{})))

			exp := c.Exporters[commonconf.UrlTemplatizationExporter].(config.GenericMap)
			assert.Equal(t, "odigos-cache."+ns+":6379", exp["endpoint"])
			assert.Equal(t, k8sconsts.OdigosCacheEndpoint(ns), exp["endpoint"])
		})
	}
}

func TestAddUrlTemplatizationUnmatchedExporter_LeavesTheRestOfTheGatewayAlone(t *testing.T) {
	c := configWithTracesIn()
	c.Exporters = config.GenericMap{"otlp/dest1": config.GenericMap{"endpoint": "x:4317"}}
	c.Service.Pipelines["traces/dest1"] = config.Pipeline{
		Receivers: []string{"forward/traces/dest1"}, Processors: []string{"batch"}, Exporters: []string{"otlp/dest1"},
	}
	pipelinesBefore, connectorsBefore := len(c.Service.Pipelines), len(c.Connectors)

	require.NoError(t, addUrlTemplatizationUnmatchedExporter(c, "odigos-system",
		utEnabledLearning(&common.LiveTrafficLearningConfiguration{})))

	assert.Contains(t, c.Exporters, "otlp/dest1", "destination exporter must survive")
	assert.Len(t, c.Service.Pipelines, pipelinesBefore, "the tap is an exporter, not a pipeline")
	assert.Len(t, c.Connectors, connectorsBefore, "the tap needs no connector")
	// Every span already reaches traces/in, so tapping a per-destination pipeline
	// as well would count the same unmatched path once per destination.
	assert.Equal(t, []string{"otlp/dest1"}, c.Service.Pipelines["traces/dest1"].Exporters,
		"only the root traces pipeline may be tapped")
}

// utUnsetShapes enumerates the nesting levels at which a tunable can be missing.
// Each of the three resolvers walks the same chain, so each needs its own fixture
// per level or a dropped nil check panics in production instead of defaulting.
func utUnsetShapes() map[string]*common.CardinalityControlConfiguration {
	return map[string]*common.CardinalityControlConfiguration{
		"nil cardinalityControl":    nil,
		"nil urlTemplatization":     {},
		"nil liveTrafficLearning":   utLearning(nil),
		"empty liveTrafficLearning": utEnabledLearning(&common.LiveTrafficLearningConfiguration{}),
	}
}

func TestMaxExamplePathsPerWorkload(t *testing.T) {
	for name, cc := range utUnsetShapes() {
		t.Run("default/"+name, func(t *testing.T) {
			assert.Equal(t, 5000, maxExamplePathsPerWorkload(cc))
		})
	}

	// A non-positive cap would make the enterprise exporter store nothing (or
	// everything, depending on how it reads the value), so it falls back instead.
	for _, tc := range []struct {
		name  string
		given int
		want  int
	}{
		{"negative falls back", -5, 5000},
		{"zero falls back", 0, 5000},
		{"one is honoured", 1, 1},
		{"explicit cap is honoured", 7000, 7000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, maxExamplePathsPerWorkload(
				utLearning(&common.LiveTrafficLearningConfiguration{MaxExamplePathsPerWorkload: utIntPtr(tc.given)})))
		})
	}
}

func TestPathExampleIdleTTL(t *testing.T) {
	for name, cc := range utUnsetShapes() {
		t.Run("default/"+name, func(t *testing.T) {
			assert.Equal(t, "48h", pathExampleIdleTTL(cc))
		})
	}

	// The exporter is handed the raw string, so the only values that may pass
	// through are the ones Go's own parser agrees with. Anything else has to
	// become the default here, because the exporter has no second chance to
	// reinterpret it.
	for _, tc := range []struct {
		name  string
		given string
		want  string
	}{
		{"valid duration passes through verbatim", "1h30m", "1h30m"},
		{"sub-hour units pass through", "90m", "90m"},
		{"fractional units pass through", "0.5h", "0.5h"},
		{"a different spelling of the default is kept", "2880m", "2880m"},
		{"unparsable falls back", "forever", "48h"},
		{"days are not a Go duration unit", "2d", "48h"},
		{"zero falls back", "0s", "48h"},
		{"negative falls back", "-1h", "48h"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pathExampleIdleTTL(
				utLearning(&common.LiveTrafficLearningConfiguration{PathExampleIdleTTL: tc.given})))
		})
	}
}

// pathExampleIdleTTL must hand the collector a duration STRING, so it cannot
// forward urltemplate.ResolvePathExampleIdleTTL's time.Duration and spells the
// fallback out as "48h" instead. Nothing in the type system ties the two
// together, so changing the shared default would leave the gateway exporter
// expiring path examples on the old window while every other consumer of
// urltemplate moved to the new one - and the quota would silently stop freeing up.
func TestResolvedDefaultsAgreeWithTheSharedUrlTemplateDefaults(t *testing.T) {
	parsed, err := time.ParseDuration(pathExampleIdleTTL(nil))
	require.NoError(t, err, "the fallback must be a duration the collector can parse")
	assert.Equal(t, urltemplate.DefaultPathExampleIdleTTL, parsed,
		"the gateway's hardcoded TTL fallback drifted from urltemplate.DefaultPathExampleIdleTTL")

	assert.Equal(t, urltemplate.DefaultMaxExamplePathsPerWorkload, maxExamplePathsPerWorkload(nil),
		"the gateway's path cap fallback drifted from urltemplate.DefaultMaxExamplePathsPerWorkload")
}

func TestLiveTrafficLearningConfig(t *testing.T) {
	assert.Nil(t, liveTrafficLearningConfig(nil))
	assert.Nil(t, liveTrafficLearningConfig(&common.CardinalityControlConfiguration{}))
	assert.Nil(t, liveTrafficLearningConfig(utLearning(nil)))

	ltl := &common.LiveTrafficLearningConfiguration{LearningInterval: "15s"}
	assert.Same(t, ltl, liveTrafficLearningConfig(utLearning(ltl)))
}

// utTracesDestination is the minimum that makes pipelinegen build a root traces
// pipeline, which is what the side channel taps. Without it the enterprise half
// of the tier table below would pass vacuously.
func utTracesDestination() *odigosv1.DestinationList {
	return &odigosv1.DestinationList{Items: []odigosv1.Destination{{
		ObjectMeta: metav1.ObjectMeta{Name: "dest1", Namespace: "odigos-system"},
		Spec: odigosv1.DestinationSpec{
			Type:            "otlp",
			DestinationName: "dest1",
			Data:            map[string]string{"OTLP_GRPC_ENDPOINT": "collector.example.com:4317"},
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}}}
}

// utGatewayConfig renders a gateway config the way syncConfigMap does for a
// cluster with one traces destination and live traffic learning turned on in the
// odigos configuration. The tier gate is applied where syncConfigMap applies it -
// around the config, not around the call - which is the part worth pinning.
func utGatewayConfig(t *testing.T, tier common.OdigosTier) *config.Config {
	t.Helper()

	cardinalityControlCfg := effectiveCardinalityControl(
		utEnabledLearning(&common.LiveTrafficLearningConfiguration{}), tier)

	ext := k8sconsts.OdigosConfigK8sExtensionType
	tailSampling := false
	gatewayOptions := pipelinegen.GatewayConfigOptions{
		OdigosNamespace:           "odigos-system",
		OdigosConfigExtensionName: &ext,
		TailSamplingEnabled:       &tailSampling,
	}

	cfg, err, status, signals := pipelinegen.CalculateGatewayConfig(
		commonconf.ToExporterConfigurerArray(utTracesDestination()), nil,
		func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error {
			if err := addSelfTelemetryPipeline(c, 8888, destinationPipelineNames, signalsRootPipelines); err != nil {
				return err
			}
			return addUrlTemplatizationUnmatchedExporter(c, "odigos-system", cardinalityControlCfg)
		},
		nil, &gatewayOptions)
	require.NoError(t, err)
	require.NoError(t, status.Destination["dest1"])
	require.Contains(t, signals, common.TracesObservabilitySignal,
		"the fixture must actually produce a traces pipeline for the tap to attach to")
	return cfg
}

// syncConfigMap calls addUrlTemplatizationUnmatchedExporter outside the
// tier.IsEnterprise() block that guards profiling and the enterprise auth
// extension, relying entirely on effectiveCardinalityControl having nil'd the
// config first. Dropping that one wrapper and passing odigosCfg.CardinalityControl
// straight through still compiles, and would install an enterprise-only exporter
// into every community gateway.
func TestGatewayConfig_UnmatchedPathExporterGatedByTier(t *testing.T) {
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)

	enterprise := utGatewayConfig(t, common.OnPremOdigosTier)
	assert.Contains(t, enterprise.Exporters, commonconf.UrlTemplatizationExporter)
	assert.Contains(t, enterprise.Service.Pipelines[rootName].Exporters, commonconf.UrlTemplatizationExporter)

	community := utGatewayConfig(t, common.CommunityOdigosTier)
	assert.NotContains(t, community.Exporters, commonconf.UrlTemplatizationExporter)
	for name, pipeline := range community.Service.Pipelines {
		assert.NotContains(t, pipeline.Exporters, commonconf.UrlTemplatizationExporter,
			"pipeline %q must not reference the enterprise exporter on community tier", name)
	}
}

func TestGatewayConfig_UnmatchedPathExporterOnlyTapsTheRootTracesPipeline(t *testing.T) {
	cfg := utGatewayConfig(t, common.OnPremOdigosTier)
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)

	tapped := []string{}
	for name, pipeline := range cfg.Service.Pipelines {
		// Every pipeline needs an exporter or the collector refuses to start.
		assert.NotEmpty(t, pipeline.Exporters, "pipeline %q has no exporters", name)
		if slices.Contains(pipeline.Exporters, commonconf.UrlTemplatizationExporter) {
			tapped = append(tapped, name)
		}
	}
	assert.Equal(t, []string{rootName}, tapped,
		"unmatched paths must be counted once, at the root, not once per destination")
}
