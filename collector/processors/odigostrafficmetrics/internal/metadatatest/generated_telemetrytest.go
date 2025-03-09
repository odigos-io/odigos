// Code generated by mdatagen. DO NOT EDIT.

package metadatatest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

func NewSettings(tt *componenttest.Telemetry) processor.Settings {
	set := processortest.NewNopSettings(processortest.NopType)
	set.ID = component.NewID(component.MustNewType("odigostrafficmetrics"))
	set.TelemetrySettings = tt.NewTelemetrySettings()
	return set
}

func AssertEqualOdigosAcceptedLogRecords(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_odigos_accepted_log_records",
		Description: "Number of log records passed through the processor.",
		Unit:        "{records}",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_odigos_accepted_log_records")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualOdigosAcceptedMetricPoints(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_odigos_accepted_metric_points",
		Description: "Number of data points passed through the processor.",
		Unit:        "{datapoints}",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_odigos_accepted_metric_points")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualOdigosAcceptedSpans(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_odigos_accepted_spans",
		Description: "Number of spans passed through the processor.",
		Unit:        "{spans}",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_odigos_accepted_spans")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualOdigosLogDataSize(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_odigos_log_data_size",
		Description: "Total size of log data passed to the processor",
		Unit:        "By",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_odigos_log_data_size")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualOdigosMetricDataSize(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_odigos_metric_data_size",
		Description: "Total size of metric data passed to the processor",
		Unit:        "By",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_odigos_metric_data_size")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}

func AssertEqualOdigosTraceDataSize(t *testing.T, tt *componenttest.Telemetry, dps []metricdata.DataPoint[int64], opts ...metricdatatest.Option) {
	want := metricdata.Metrics{
		Name:        "otelcol_odigos_trace_data_size",
		Description: "Total size of trace data passed to the processor",
		Unit:        "By",
		Data: metricdata.Sum[int64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints:  dps,
		},
	}
	got, err := tt.GetMetric("otelcol_odigos_trace_data_size")
	require.NoError(t, err)
	metricdatatest.AssertEqual(t, want, got, opts...)
}
