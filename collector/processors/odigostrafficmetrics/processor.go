package odigostrafficmetrics

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigostrafficmetrics/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type dataSizesMetricsProcessor struct {
	logger                  *zap.Logger
	tracesSizer             *ptrace.ProtoMarshaler
	metricsSizer            *pmetric.ProtoMarshaler
	logsSizer               *plog.ProtoMarshaler
	resAttrsKeys            []string
	samplingFraction        float64
	inverseSamplingFraction int64

	obsrep *metadata.TelemetryBuilder
}

func newThroughputMeasurementProcessor(set processor.Settings, cfg *Config) (*dataSizesMetricsProcessor, error) {
	samplingFraction := cfg.SamplingRatio
	var inverseSamplingFraction int64
	if samplingFraction != 0 {
		inverseSamplingFraction = int64(1 / samplingFraction)
	}

	set.Logger.Info("Odigos traffic metrics processor is enabled with the following configuration",
		zap.String("sampling_ratio", fmt.Sprintf("%f", samplingFraction)),
		zap.String("inverse_sampling_ratio", fmt.Sprintf("%d", inverseSamplingFraction)),
	)

	obsrep, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	return &dataSizesMetricsProcessor{
		logger:                  set.Logger,
		tracesSizer:             &ptrace.ProtoMarshaler{},
		metricsSizer:            &pmetric.ProtoMarshaler{},
		logsSizer:               &plog.ProtoMarshaler{},
		resAttrsKeys:            cfg.ResourceAttributesKeys,
		samplingFraction:        samplingFraction,
		inverseSamplingFraction: inverseSamplingFraction,
		obsrep:                  obsrep,
	}, nil
}

func (p *dataSizesMetricsProcessor) attributeSetFromResource(res pcommon.Resource) attribute.Set {
	result := []attribute.KeyValue{}
	attrs := res.Attributes()
	for _, key := range p.resAttrsKeys {
		if v, ok := attrs.Get(key); ok {
			result = append(result, attribute.String(key, v.Str()))
		}
	}
	return attribute.NewSet(result...)
}

func (p *dataSizesMetricsProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if p.samplingFraction != 0 && rand.Float64() < p.samplingFraction {
		resSpans := td.ResourceSpans()
		for i := 0; i < resSpans.Len(); i++ {
			res := resSpans.At(i).Resource()
			p.obsrep.OdigosTraceDataSize.Add(ctx,
				int64(p.tracesSizer.ResourceSpansSize(resSpans.At(i)))*p.inverseSamplingFraction,
				metric.WithAttributeSet(p.attributeSetFromResource(res)),
			)
		}
		p.obsrep.OdigosAcceptedSpans.Add(ctx, int64(td.SpanCount()))
	}
	return td, nil
}

func (p *dataSizesMetricsProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	if p.samplingFraction != 0 && rand.Float64() < p.samplingFraction {
		resLogs := ld.ResourceLogs()
		for i := 0; i < resLogs.Len(); i++ {
			res := resLogs.At(i).Resource()
			p.obsrep.OdigosLogDataSize.Add(ctx,
				int64(p.logsSizer.ResourceLogsSize(resLogs.At(i)))*p.inverseSamplingFraction,
				metric.WithAttributeSet(p.attributeSetFromResource(res)),
			)
		}
		p.obsrep.OdigosAcceptedLogRecords.Add(ctx, int64(ld.LogRecordCount()))
	}
	return ld, nil
}

func (p *dataSizesMetricsProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if p.samplingFraction != 0 && rand.Float64() < p.samplingFraction {
		resMetrics := md.ResourceMetrics()
		for i := 0; i < resMetrics.Len(); i++ {
			res := resMetrics.At(i).Resource()
			p.obsrep.OdigosMetricDataSize.Add(ctx, 
				int64(p.metricsSizer.ResourceMetricsSize(resMetrics.At(i)))*p.inverseSamplingFraction,
				metric.WithAttributeSet(p.attributeSetFromResource(res)),
			)
		}
		p.obsrep.OdigosAcceptedMetricPoints.Add(ctx, int64(md.MetricCount()))
	}
	return md, nil
}
