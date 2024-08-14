package odigostrafficmetrics

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigostrafficmetrics/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type dataSizesMetricsProcessor struct {
	logger                                     *zap.Logger
	logSize, metricSize, traceSize             metric.Int64Counter
	tracesSizer                                ptrace.Sizer
	metricsSizer                               pmetric.Sizer
	logsSizer                                  plog.Sizer
	resAttrsKeys                               []string
	samplingFraction                           float64
	inverseSamplingFraction                    int64

	obsrep     *processorhelper.ObsReport
}

func newThroughputMeasurementProcessor(set processor.Settings, mp metric.MeterProvider, cfg *Config) (*dataSizesMetricsProcessor, error) {
	meter := mp.Meter("github.com/odigos-io/odigos/collector/processors/odigostrafficmetrics")

	logSize, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(metadata.Type.String(), "log_data_size"),
		metric.WithDescription("Total size of log data passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create log_data_size counter: %w", err)
	}

	metricSize, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(metadata.Type.String(), "metric_data_size"),
		metric.WithDescription("Total size of metric data passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create metric_data_size counter: %w", err)
	}

	traceSize, err := meter.Int64Counter(
		processorhelper.BuildCustomMetricName(metadata.Type.String(), "trace_data_size"),
		metric.WithDescription("Total size of trace data passed to the processor"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace_data_size counter: %w", err)
	}

	samplingFraction := cfg.SamplingRatio
	var inverseSamplingFraction int64
	if samplingFraction != 0 {
		inverseSamplingFraction = int64(1 / samplingFraction)
	}

	set.Logger.Info("Odigos traffic metrics processor is enabled with the following configuration", 
		zap.String("sampling_ratio", fmt.Sprintf("%f", samplingFraction)),
		zap.String("inverse_sampling_ratio", fmt.Sprintf("%d", inverseSamplingFraction)),
	)

	obsrep, err := processorhelper.NewObsReport(processorhelper.ObsReportSettings{
		ProcessorID:             set.ID,
		ProcessorCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	return &dataSizesMetricsProcessor{
		logger:                  set.Logger,
		logSize:                 logSize,
		metricSize:              metricSize,
		traceSize:               traceSize,
		tracesSizer:             &ptrace.ProtoMarshaler{},
		metricsSizer:            &pmetric.ProtoMarshaler{},
		logsSizer:               &plog.ProtoMarshaler{},
		resAttrsKeys:            cfg.ResourceAttributesKeys,
		samplingFraction:        samplingFraction,
		inverseSamplingFraction: inverseSamplingFraction,
		obsrep: 				 obsrep,
	}, nil
}

func (p *dataSizesMetricsProcessor) traceAttributes(td ptrace.Traces) []attribute.KeyValue {
	resSpans := td.ResourceSpans()
	result := []attribute.KeyValue{}
	for i := 0; i < resSpans.Len(); i++ {
		res := resSpans.At(i).Resource()
		attrs := res.Attributes()
		for _, key := range p.resAttrsKeys {
			if v, ok := attrs.Get(key); ok {
				result = append(result, attribute.String(key, v.Str()))
			}
		}
	}
	return result
}

func (p *dataSizesMetricsProcessor) logAttributes(ld plog.Logs) []attribute.KeyValue {
	resSpans := ld.ResourceLogs()
	result := []attribute.KeyValue{}
	for i := 0; i < resSpans.Len(); i++ {
		res := resSpans.At(i).Resource()
		attrs := res.Attributes()
		for _, key := range p.resAttrsKeys {
			if v, ok := attrs.Get(key); ok {
				result = append(result, attribute.String(key, v.Str()))
			}
		}
	}
	return result
}

func (p *dataSizesMetricsProcessor) meterAttributes(md pmetric.Metrics) []attribute.KeyValue {
	resSpans := md.ResourceMetrics()
	result := []attribute.KeyValue{}
	for i := 0; i < resSpans.Len(); i++ {
		res := resSpans.At(i).Resource()
		attrs := res.Attributes()
		for _, key := range p.resAttrsKeys {
			if v, ok := attrs.Get(key); ok {
				result = append(result, attribute.String(key, v.Str()))
			}
		}
	}
	return result
}

func (p *dataSizesMetricsProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if p.samplingFraction != 0 && rand.Float64() < p.samplingFraction {
		p.traceSize.Add(ctx, int64(p.tracesSizer.TracesSize(td)) * p.inverseSamplingFraction, metric.WithAttributes(p.traceAttributes(td)...))
	}
	p.obsrep.TracesAccepted(ctx, td.SpanCount())
	return td, nil
}

func (p *dataSizesMetricsProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if p.samplingFraction != 0 && rand.Float64() < p.samplingFraction {
		p.logSize.Add(ctx, int64(p.logsSizer.LogsSize(ld)) * p.inverseSamplingFraction, metric.WithAttributes(p.logAttributes(ld)...))
	}
	p.obsrep.LogsAccepted(ctx, ld.LogRecordCount())
	return ld, nil
}

func (p *dataSizesMetricsProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if p.samplingFraction != 0 && rand.Float64() < p.samplingFraction {
		p.metricSize.Add(ctx, int64(p.metricsSizer.MetricsSize(md)) * p.inverseSamplingFraction, metric.WithAttributes(p.meterAttributes(md)...))
	}
	p.obsrep.MetricsAccepted(ctx, md.MetricCount())
	return md, nil
}
