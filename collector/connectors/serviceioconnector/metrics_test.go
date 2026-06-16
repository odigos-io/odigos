package serviceioconnector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func TestBuildMetrics(t *testing.T) {
	connector := &serviceioConnector{
		config:              &Config{},
		startTime:           time.Unix(1700000000, 0),
		keyToMetric:         make(map[uint64]metricSeries),
		collectorInstanceID: "odigos-gateway-test-pod",
	}

	inputAttrs := pcommon.NewMap()
	inputAttrs.PutStr(inputAttributePrefix+"http.route", "/users")
	outputAttrs := pcommon.NewMap()
	outputAttrs.PutStr(outputAttributePrefix+"rpc.service", "UserService")
	instance := &ServiceInstance{
		ServiceName:        "svc-1",
		ResourceAttributes: pcommon.NewMap(),
	}
	inputAttributes := buildServiceInstanceBaseAttributes(instance, inputAttrs)
	key, attributes := buildConnectionAttributes(inputAttributes, outputAttrs)
	now := time.Now()
	connector.keyToMetric[key] = metricSeries{
		dimensions:  attributes,
		resource:    buildMetricResourceAttributes(instance),
		count:       3,
		startTime:   connector.startTime,
		lastUpdated: now,
	}

	md, err := connector.buildMetrics()
	require.NoError(t, err)
	require.Equal(t, 1, md.MetricCount())

	metric := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0)
	require.Equal(t, metricNameConnectionTotal, metric.Name())
	require.Equal(t, pmetric.AggregationTemporalityCumulative, metric.Sum().AggregationTemporality())

	dp := metric.Sum().DataPoints().At(0)
	require.EqualValues(t, 3, dp.IntValue())
	require.Equal(t, pcommon.NewTimestampFromTime(connector.startTime), dp.StartTimestamp())
	require.True(t, dp.Timestamp() > 0)

	serviceName, ok := dp.Attributes().Get(string(semconv.ServiceNameKey))
	require.True(t, ok)
	require.Equal(t, "svc-1", serviceName.Str())

	instanceID, ok := dp.Attributes().Get(collectorInstanceAttributeId)
	require.True(t, ok)
	require.Equal(t, "odigos-gateway-test-pod", instanceID.Str())
}

func TestBuildMetricsPrunesStaleSeries(t *testing.T) {
	connector := &serviceioConnector{
		config:      &Config{},
		startTime:   time.Unix(1700000000, 0),
		keyToMetric: make(map[uint64]metricSeries),
	}

	freshAttrs := pcommon.NewMap()
	freshAttrs.PutStr("service.name", "checkout")
	freshKey := hashAttributes(freshAttrs)
	connector.keyToMetric[freshKey] = metricSeries{
		dimensions:  freshAttrs,
		resource:    pcommon.NewMap(),
		count:       5,
		startTime:   connector.startTime,
		lastUpdated: time.Now(),
	}

	staleAttrs := pcommon.NewMap()
	staleAttrs.PutStr("service.name", "orders")
	staleKey := hashAttributes(staleAttrs)
	connector.keyToMetric[staleKey] = metricSeries{
		dimensions:  staleAttrs,
		resource:    pcommon.NewMap(),
		count:       7,
		startTime:   connector.startTime,
		lastUpdated: time.Now().Add(-(metricSeriesTTL + time.Minute)),
	}

	md, err := connector.buildMetrics()
	require.NoError(t, err)
	require.Equal(t, 1, md.MetricCount())
	require.Contains(t, connector.keyToMetric, freshKey)
	require.NotContains(t, connector.keyToMetric, staleKey)

	dps := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().At(0).Sum().DataPoints()
	require.Equal(t, 1, dps.Len())
	serviceName, ok := dps.At(0).Attributes().Get("service.name")
	require.True(t, ok)
	require.Equal(t, "checkout", serviceName.Str())
}

func TestConnectionAttributes_IsDeterministic(t *testing.T) {
	input1 := pcommon.NewMap()
	input1.PutStr("input.b", "2")
	input1.PutStr("input.a", "1")
	input2 := pcommon.NewMap()
	input2.PutStr("input.a", "1")
	input2.PutStr("input.b", "2")
	key1, _ := buildConnectionAttributes(buildServiceInstanceBaseAttributes(&ServiceInstance{ResourceAttributes: pcommon.NewMap()}, input1), pcommon.NewMap())
	key2, _ := buildConnectionAttributes(buildServiceInstanceBaseAttributes(&ServiceInstance{ResourceAttributes: pcommon.NewMap()}, input2), pcommon.NewMap())
	require.Equal(t, key1, key2)
}

func TestHashAttributes_DifferentValuesProduceDifferentKeys(t *testing.T) {
	attrs1 := pcommon.NewMap()
	attrs1.PutStr("a", "1")
	attrs2 := pcommon.NewMap()
	attrs2.PutStr("a", "2")
	require.NotEqual(t, hashAttributes(attrs1), hashAttributes(attrs2))
}
