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

type resSpanMock struct {
	serviceName, spanName string
	resAttrs              map[string]any
}

func generateTraceData(resourceSpans ...resSpanMock) ptrace.Traces {
	td := ptrace.NewTraces()
	for _, resMock := range resourceSpans {
		rs := td.ResourceSpans().AppendEmpty()
		if resMock.serviceName != "" {
			rs.Resource().Attributes().FromRaw(resMock.resAttrs)
			rs.Resource().Attributes().PutStr(string(semconv.ServiceNameKey), resMock.serviceName)
		}
		span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		span.SetName(resMock.serviceName)
	}
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

	set := processortest.NewNopSettings(processortest.NopType)
	set.MeterProvider = metricProvider

	tmp, err := newThroughputMeasurementProcessor(set, &Config{
		ResourceAttributesKeys: []string{"service.name", "key1_1", "key1_2", "key2_1", "key2_2"},
		SamplingRatio:          1,
	})
	require.NoError(t, err)

	traces := generateTraceData(
		resSpanMock{
			serviceName: "service-name1",
			spanName:    "span-name1",
			resAttrs: map[string]any{
				"key1_1": "val1_1",
				"key1_2": "val1_2",
			},
		},
		resSpanMock{
			serviceName: "service-name2",
			spanName:    "span-name2",
			resAttrs: map[string]any{
				"key2_1": "val2_1",
				"key2_2": "val2_2",
			},
		},
	)
	require.NoError(t, err)

	processedTraces, err := tmp.processTraces(context.Background(), traces)
	require.NoError(t, err)

	// Output traces should be the same as input traces (passthrough check)
	require.NoError(t, ptracetest.CompareTraces(traces, processedTraces))

	var rm metricdata.ResourceMetrics
	require.NoError(t, metricsReader.Collect(context.Background(), &rm))

	var traceSize int64
	require.Greater(t, len(rm.ScopeMetrics), 0)

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "otelcol_odigos_trace_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 2, len(sum.DataPoints))

				for i := range 2 {
					attrs := sum.DataPoints[i].Attributes.ToSlice()
					var serviceName, key1, key2, val1, val2 string
					for _, attr := range attrs {
						switch attr.Key {
						case "service.name":
							serviceName = attr.Value.AsString()
						case "key1_1":
							key1 = "key1_1"
							val1 = attr.Value.AsString()
						case "key1_2":
							key2 = "key1_2"
							val2 = attr.Value.AsString()
						case "key2_1":
							key1 = "key2_1"
							val1 = attr.Value.AsString()
						case "key2_2":
							key2 = "key2_2"
							val2 = attr.Value.AsString()
						default:
							t.Errorf("unexpected attribute key: %s", attr.Key)
						}
					}

					switch serviceName {
					case "service-name1":
						require.Equal(t, key1, "key1_1")
						require.Equal(t, key2, "key1_2")
						require.Equal(t, val1, "val1_1")
						require.Equal(t, val2, "val1_2")
					case "service-name2":
						require.Equal(t, key1, "key2_1")
						require.Equal(t, key2, "key2_2")
						require.Equal(t, val1, "val2_1")
						require.Equal(t, val2, "val2_2")
					}

					require.Greater(t, sum.DataPoints[i].Value, int64(0))
					traceSize += sum.DataPoints[i].Value
				}
			}
		}
	}

	_, err = tmp.processTraces(context.Background(), traces)
	require.NoError(t, err)
	require.NoError(t, metricsReader.Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, metric := range sm.Metrics {
			switch metric.Name {
			case "otelcol_odigos_trace_data_size":
				sum := metric.Data.(metricdata.Sum[int64])
				require.Equal(t, 2, len(sum.DataPoints))
				secondTraceCounterVal := int64(0)
				for i := range 2 {
					secondTraceCounterVal += sum.DataPoints[i].Value
				}

				require.Equal(t, traceSize*2, secondTraceCounterVal)
			}
		}
	}
}
