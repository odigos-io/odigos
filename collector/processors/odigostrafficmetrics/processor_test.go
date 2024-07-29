package odigostrafficmetrics

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/ptracetest"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func generateTraceData(serviceName, spanName string, resAttrs map[string]any) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	if serviceName != "" {
		rs.Resource().Attributes().FromRaw(resAttrs)
		rs.Resource().Attributes().PutStr(string(semconv.ServiceNameKey), serviceName)
	}
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName(spanName)
	return td
}

func TestProcessor_Logs(t *testing.T) {

}

func TestProcessor_Metrics(t *testing.T) {

}

func TestProcessor_Traces(t *testing.T) {
	metricsReader := metric.NewManualReader()
	defer metricsReader.Shutdown(context.Background())

	metricProvider := metric.NewMeterProvider(
		metric.WithReader(metricsReader),
	)
	defer metricProvider.Shutdown(context.Background())

	set := processortest.NewNopSettings()

	tmp, err := newThroughputMeasurementProcessor(set, metricProvider, &Config{
		ResourceAttributesKeys: []string{"service.name", "key1"},
		SamplingRatio: 		     1,
	})
	require.NoError(t, err)

	traces := generateTraceData("service-name", "span-name", map[string]any{
		"key1": "value1",
		"key2": "value2",
	})
	require.NoError(t, err)

	processedTraces, err := tmp.processTraces(context.Background(), traces)
	require.NoError(t, err)

	// Output traces should be the same as input traces (passthrough check)
	require.NoError(t, ptracetest.CompareTraces(traces, processedTraces))

	var rm metricdata.ResourceMetrics
	require.NoError(t, metricsReader.Collect(context.Background(), &rm))

	var firstTraceSize int64
	require.Greater(t, len(rm.ScopeMetrics), 0)

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "processor_odigostrafficmetrics_trace_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				attrs := sum.DataPoints[0].Attributes.ToSlice()
				for _, attr := range attrs {
					switch attr.Key {
					case "service.name":
						require.Equal(t, "service-name", attr.Value.AsString())
					case "key1":
						require.Equal(t, "value1", attr.Value.AsString())
					default:
						t.Errorf("unexpected attribute key: %s", attr.Key)
					}
				}

				firstTraceSize = sum.DataPoints[0].Value
				require.Greater(t, firstTraceSize, int64(0))
			}
		}
	}

	_, err = tmp.processTraces(context.Background(), traces)
	require.NoError(t, err)
	require.NoError(t, metricsReader.Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "processor_odigostrafficmetrics_trace_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 1, len(sum.DataPoints))

				attrs := sum.DataPoints[0].Attributes.ToSlice()
				for _, attr := range attrs {
					switch attr.Key {
					case "service.name":
						require.Equal(t, "service-name", attr.Value.AsString())
					case "key1":
						require.Equal(t, "value1", attr.Value.AsString())
					default:
						t.Errorf("unexpected attribute key: %s", attr.Key)
					}
				}

				secondTraceCounterVal := sum.DataPoints[0].Value
				require.Equal(t, firstTraceSize*2, secondTraceCounterVal)
			}
		}
	}

}
