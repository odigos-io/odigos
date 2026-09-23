package metrics

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	controllermetric "sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	mpNodeName = "ip-10-0-1-7.ec2.internal"
	mpPodName  = "odiglet-jk4mz"
	mpService  = "odiglet"
	// the prometheus rendering of the k8s.node.name resource attribute, which is what dashboards
	// and alerts select on.
	mpNodeLabel = "k8s_node_name"
)

// The exporter registers itself on the process-wide controller-runtime registry and there is no way
// to unregister it, so a provider is built at most once per test binary. Building one per test (or
// per -count iteration) would export the same series twice and make Gather fail.
var (
	mpProviderOnce sync.Once
	mpProvider     metric.MeterProvider
	mpProviderErr  error

	mpNodelessProviderOnce sync.Once
	mpNodelessProvider     metric.MeterProvider
	mpNodelessProviderErr  error
)

// mpFullResource carries the node name alongside two attributes that must NOT become labels.
func mpFullResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SNodeName(mpNodeName),
		semconv.ServiceName(mpService),
		semconv.K8SPodName(mpPodName),
	)
}

func mpSharedProvider(t *testing.T) metric.MeterProvider {
	t.Helper()
	mpProviderOnce.Do(func() {
		mpProvider, mpProviderErr = NewMeterProviderForController(mpFullResource())
	})
	require.NoError(t, mpProviderErr)
	require.NotNil(t, mpProvider)
	return mpProvider
}

func mpNodelessSharedProvider(t *testing.T) metric.MeterProvider {
	t.Helper()
	mpNodelessProviderOnce.Do(func() {
		mpNodelessProvider, mpNodelessProviderErr = NewMeterProviderForController(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("collector-without-a-node-name"),
		))
	})
	require.NoError(t, mpNodelessProviderErr)
	require.NotNil(t, mpNodelessProvider)
	return mpNodelessProvider
}

type mpSeries struct {
	labels  map[string]string
	counter float64
	gauge   float64
}

// mpGather reads the controller-runtime registry the provider is expected to export through, and
// flattens it into plain types keyed by metric family name.
func mpGather(t *testing.T) map[string][]mpSeries {
	t.Helper()
	families, err := controllermetric.Registry.Gather()
	require.NoError(t, err)

	gathered := make(map[string][]mpSeries, len(families))
	for _, family := range families {
		for _, m := range family.GetMetric() {
			labels := make(map[string]string, len(m.GetLabel()))
			for _, label := range m.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			gathered[family.GetName()] = append(gathered[family.GetName()], mpSeries{
				labels:  labels,
				counter: m.GetCounter().GetValue(),
				gauge:   m.GetGauge().GetValue(),
			})
		}
	}
	return gathered
}

func mpOnlySeries(t *testing.T, familyName string) mpSeries {
	t.Helper()
	series := mpGather(t)[familyName]
	require.Len(t, series, 1, "expected exactly one series for %s", familyName)
	return series[0]
}

func mpAddCounter(t *testing.T, provider metric.MeterProvider, name string, opts ...metric.AddOption) {
	t.Helper()
	counter, err := provider.Meter("k8sutils/pkg/metrics_test").Int64Counter(name)
	require.NoError(t, err)
	counter.Add(context.Background(), 1, opts...)
}

func TestTheMeterProviderExportsThroughTheControllerRuntimeRegistry(t *testing.T) {
	mpAddCounter(t, mpSharedProvider(t), "mp_registry_total")

	series := mpOnlySeries(t, "mp_registry_total")
	assert.Positive(t, series.counter)
}

func TestTheNodeNameResourceAttributeBecomesAConstantLabel(t *testing.T) {
	mpAddCounter(t, mpSharedProvider(t), "mp_node_label_total")

	series := mpOnlySeries(t, "mp_node_label_total")
	assert.Equal(t, mpNodeName, series.labels[mpNodeLabel])
}

func TestOnlyTheNodeNameResourceAttributeBecomesAConstantLabel(t *testing.T) {
	mpAddCounter(t, mpSharedProvider(t), "mp_allow_filter_total")

	series := mpOnlySeries(t, "mp_allow_filter_total")
	assert.NotContains(t, series.labels, "service_name")
	assert.NotContains(t, series.labels, "k8s_pod_name")

	// anti-vacuity: the two attributes really are on the resource, they are only kept off the
	// individual series. target_info is where the full resource is published.
	var targetInfo map[string]string
	for _, candidate := range mpGather(t)["target_info"] {
		if candidate.labels[mpNodeLabel] == mpNodeName {
			targetInfo = candidate.labels
		}
	}
	require.NotNil(t, targetInfo, "the provider did not publish a target_info series for this resource")
	assert.Equal(t, mpService, targetInfo["service_name"])
	assert.Equal(t, mpPodName, targetInfo["k8s_pod_name"])
}

func TestTheConstantLabelIsAddedToEverySeriesWithoutDroppingInstrumentAttributes(t *testing.T) {
	provider := mpSharedProvider(t)
	mpAddCounter(t, provider, "mp_per_series_total", metric.WithAttributes(attribute.String("map_type", "Hash")))
	mpAddCounter(t, provider, "mp_per_series_total", metric.WithAttributes(attribute.String("map_type", "RingBuf")))

	byMapType := make(map[string]map[string]string)
	for _, series := range mpGather(t)["mp_per_series_total"] {
		byMapType[series.labels["map_type"]] = series.labels
	}

	require.Len(t, byMapType, 2)
	for mapType, labels := range byMapType {
		assert.Equal(t, mpNodeName, labels[mpNodeLabel], "series map_type=%s lost the node name", mapType)
	}
}

func TestAResourceWithoutTheNodeNameProducesNoNodeLabel(t *testing.T) {
	mpAddCounter(t, mpNodelessSharedProvider(t), "mp_no_node_total")

	series := mpOnlySeries(t, "mp_no_node_total")
	assert.NotContains(t, series.labels, mpNodeLabel)
}

// The semconv key and its prometheus rendering are the contract dashboards select on; nothing links
// them at compile time.
func TestTheNodeNameLabelIsThePrometheusRenderingOfTheSemconvKey(t *testing.T) {
	assert.Equal(t, "k8s.node.name", string(semconv.K8SNodeNameKey))

	mpAddCounter(t, mpSharedProvider(t), "mp_label_name_total")

	series := mpOnlySeries(t, "mp_label_name_total")
	assert.Contains(t, series.labels, mpNodeLabel)
}
