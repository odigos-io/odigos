package odigostrafficmetrics

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigostrafficmetrics/internal/metadata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
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
}

func newThroughputMeasurementProcessor(logger *zap.Logger, mp metric.MeterProvider, cfg *Config) (*dataSizesMetricsProcessor, error) {
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

	return &dataSizesMetricsProcessor{
		logger:         logger,
		logSize:        logSize,
		metricSize:     metricSize,
		traceSize:      traceSize,
		tracesSizer:    &ptrace.ProtoMarshaler{},
		metricsSizer:   &pmetric.ProtoMarshaler{},
		logsSizer:      &plog.ProtoMarshaler{},
		resAttrsKeys:   cfg.ResourceAttributesKeys,
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
	p.traceSize.Add(ctx, int64(p.tracesSizer.TracesSize(td)), metric.WithAttributes(p.traceAttributes(td)...))
	return td, nil
}

func (p *dataSizesMetricsProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	p.logSize.Add(ctx, int64(p.logsSizer.LogsSize(ld)), metric.WithAttributes(p.logAttributes(ld)...))
	return ld, nil
}

func (p *dataSizesMetricsProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	p.metricSize.Add(ctx, int64(p.metricsSizer.MetricsSize(md)), metric.WithAttributes(p.meterAttributes(md)...))
	return md, nil
}
