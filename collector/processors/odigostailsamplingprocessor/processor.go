package odigostailsamplingprocessor

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/pkg/completetrace"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/config"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/costreduction"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/highlyrelevant"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/metrics"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/noisy"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/samplingspanattrs"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/internal/metadata"
	"github.com/odigos-io/odigos/common/consts"
)

type tailSamplingProcessor struct {
	logger      *zap.Logger
	config      *Config
	configCache *config.ConfigCache

	noisyOperationsCategoryMeasurementOptions metric.MeasurementOption
	highlyRelevantCategoryMeasurementOptions  metric.MeasurementOption
	costReductionCategoryMeasurementOptions   metric.MeasurementOption

	telemetryBuilder *metadata.TelemetryBuilder
}

func (p *tailSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {

	if !p.configCache.Attached() {
		p.logger.Error("odigos config extension is not set, skipping tail sampling")
		return td, nil // for auto generated tests, and not to crash in case it somehow happens
	}

	traceID, shouldProcess, spanCount, err := completetrace.ValidateCompleteTrace(td)
	if err != nil {
		p.logger.Error("failed to check prerequists", zap.Error(err))
		return td, nil
	}
	if !shouldProcess {
		return td, nil
	}

	// record that we are checking a new trace for tail sampling.
	p.recordTraceCheckMetrics(ctx, nil, spanCount)

	rnd := sampling.TraceIDToRandomness(traceID)
	// convert from range [0-MaxAdjustedCount] to range [0-100]
	tracePercentage := float64(rnd.Unsigned()) / float64(sampling.MaxAdjustedCount) * 100.0

	// Noisy operations category.
	noisyOperationRes := p.evaluateNoisyOperations(td)
	p.recordMetrics(ctx, noisyOperationRes.RulesEvalResults, tracePercentage, p.noisyOperationsCategoryMeasurementOptions)
	if noisyOperationRes.DecidingRule != nil {
		noisyOperationRule := noisyOperationRes.DecidingRule
		percentageAtMost := noisyOperationRule.Percentage
		keepTrace := tracePercentage <= percentageAtMost

		p.recordCategoryMatchMetrics(ctx, p.noisyOperationsCategoryMeasurementOptions, keepTrace, spanCount)

		if keepTrace || p.config.DryRun {
			samplingspanattrs.SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryNoise, noisyOperationRule, p.config.DryRun, keepTrace, p.config.SpanSamplingAttributes)
			return td, nil
		}
		return ptrace.NewTraces(), nil
	}

	highlyRelevantRes := highlyrelevant.Evaluate(td, p.configCache)
	p.recordMetrics(ctx, highlyRelevantRes.RulesEvalResults, tracePercentage, p.highlyRelevantCategoryMeasurementOptions)
	if highlyRelevantRes.DecidingRule != nil {
		highlyRelevantOperationRule := highlyRelevantRes.DecidingRule
		percentageAtLeast := highlyRelevantOperationRule.Percentage
		keepTrace := tracePercentage <= percentageAtLeast

		p.recordCategoryMatchMetrics(ctx, p.highlyRelevantCategoryMeasurementOptions, keepTrace, spanCount)

		if keepTrace || p.config.DryRun {
			samplingspanattrs.SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryHighlyRelevant, highlyRelevantOperationRule, p.config.DryRun, keepTrace, p.config.SpanSamplingAttributes)
			return td, nil
		}
		return ptrace.NewTraces(), nil
	}

	costReductionRes := costreduction.Evaluate(td, p.configCache)
	p.recordMetrics(ctx, costReductionRes.RulesEvalResults, tracePercentage, p.costReductionCategoryMeasurementOptions)
	if costReductionRes.DecidingRule != nil {
		costReductionRule := costReductionRes.DecidingRule
		percentageAtMost := costReductionRule.Percentage
		keepTrace := tracePercentage <= percentageAtMost

		p.recordCategoryMatchMetrics(ctx, p.costReductionCategoryMeasurementOptions, keepTrace, spanCount)

		if keepTrace || p.config.DryRun {
			samplingspanattrs.SetTraceSamplingAttributesOnSpans(td, consts.SamplingCategoryCostReduction, costReductionRule, p.config.DryRun, keepTrace, p.config.SpanSamplingAttributes)
			return td, nil
		}
		return ptrace.NewTraces(), nil
	}

	return td, nil
}

// evaluateNoisyOperations evaluates the noisy operations category for the trace.
// it return the result of the evaluation.
func (p *tailSamplingProcessor) evaluateNoisyOperations(td ptrace.Traces) noisy.NoisyOperationsEvaluationResult {

	rootSpan, resource, found := getRootSpan(td)
	if !found {
		// the root span is missing, so we cannot apply noisy operations category
		// as the rules are evaluated only on the root span.
		return noisy.NoisyOperationsEvaluationResult{
			DecidingRule:     nil,
			RulesEvalResults: nil,
		}
	}

	tailSamplingConfig, ok := p.configCache.GetTailSamplingConfig(resource)
	if !ok {
		// the tail sampling config is set only if there are actually any rules.
		// this source is not relevant for noisy operations category.
		return noisy.NoisyOperationsEvaluationResult{
			DecidingRule:     nil,
			RulesEvalResults: nil,
		}
	}

	return noisy.Evaluate(rootSpan, tailSamplingConfig.NoisyOperations)
}

func (p *tailSamplingProcessor) Start(ctx context.Context, host component.Host) error {
	return p.configCache.Start(ctx, host, p.config.OdigosConfigExtension)
}

func (p *tailSamplingProcessor) Shutdown(ctx context.Context) error {
	return p.configCache.Shutdown(ctx)
}

func (p *tailSamplingProcessor) recordMetrics(ctx context.Context, evalResult category.CategoryRulesEvaluationResults, tracePercentage float64, categoryMeasurementOptions metric.MeasurementOption) {

	// record per rule metrics.
	for _, result := range evalResult {

		rulesMeasurementOptions := metric.WithAttributeSet(result.ComputedRule.MetricsAttributes)

		p.telemetryBuilder.OdigosSamplingSpanCheckCount.Add(ctx, int64(result.SpanCheckedCount), rulesMeasurementOptions)
		p.telemetryBuilder.OdigosSamplingTraceCheckCount.Add(ctx, 1, rulesMeasurementOptions)

		if result.SpanMatchedCount > 0 {
			p.telemetryBuilder.OdigosSamplingSpanMatchCount.Add(ctx, int64(result.SpanMatchedCount), rulesMeasurementOptions)
			p.telemetryBuilder.OdigosSamplingTraceMatchCount.Add(ctx, 1, rulesMeasurementOptions)

			// for each rule, record if it is evaluated to drop or keep the trace.
			// even if disabled or in dry run mode, we can still monitor the decision.
			dropped := tracePercentage > result.ComputedRule.Percentage
			if dropped {
				p.telemetryBuilder.OdigosSamplingTraceDropCount.Add(ctx, 1, rulesMeasurementOptions)
			} else {
				p.telemetryBuilder.OdigosSamplingTraceKeepCount.Add(ctx, 1, rulesMeasurementOptions)
			}
		}
	}

	// record that a category was checked for single trace.
	p.telemetryBuilder.OdigosSamplingTraceCheckCount.Add(ctx, 1, categoryMeasurementOptions)
}

func (p *tailSamplingProcessor) recordCategoryMatchMetrics(ctx context.Context, measurementOptions metric.MeasurementOption, kept bool, spansCount int) {
	p.telemetryBuilder.OdigosSamplingTraceMatchCount.Add(ctx, 1, measurementOptions)
	p.telemetryBuilder.OdigosSamplingSpanMatchCount.Add(ctx, int64(spansCount), measurementOptions)
	if kept {
		p.telemetryBuilder.OdigosSamplingTraceKeepCount.Add(ctx, 1, measurementOptions)
		p.telemetryBuilder.OdigosSamplingSpanKeepCount.Add(ctx, int64(spansCount), measurementOptions)
	} else {
		p.telemetryBuilder.OdigosSamplingTraceDropCount.Add(ctx, 1, measurementOptions)
		p.telemetryBuilder.OdigosSamplingSpanDropCount.Add(ctx, int64(spansCount), measurementOptions)
	}
}

func (p *tailSamplingProcessor) recordTraceCheckMetrics(ctx context.Context, kept *bool, spansCount int) {
	p.telemetryBuilder.OdigosSamplingTraceCheckCount.Add(ctx, 1)
	p.telemetryBuilder.OdigosSamplingSpanCheckCount.Add(ctx, int64(spansCount))
}

func newTailSamplingProcessor(logger *zap.Logger, cfg *Config, set component.TelemetrySettings) *tailSamplingProcessor {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		logger.Error("failed to create telemetry builder", zap.Error(err))
	}

	// compute once to to avoid creating new sets on each call.
	noisyOperationsCategoryAttributes := metrics.CategoryMetricsAttributeSet(consts.SamplingCategoryNoise, cfg.DryRun)
	highlyRelevantCategoryAttributes := metrics.CategoryMetricsAttributeSet(consts.SamplingCategoryHighlyRelevant, cfg.DryRun)
	costReductionCategoryAttributes := metrics.CategoryMetricsAttributeSet(consts.SamplingCategoryCostReduction, cfg.DryRun)

	return &tailSamplingProcessor{
		logger:           logger,
		config:           cfg,
		configCache:      config.NewConfigCache(logger, cfg.DryRun),
		telemetryBuilder: telemetryBuilder,
		noisyOperationsCategoryMeasurementOptions: metric.WithAttributeSet(noisyOperationsCategoryAttributes),
		highlyRelevantCategoryMeasurementOptions:  metric.WithAttributeSet(highlyRelevantCategoryAttributes),
		costReductionCategoryMeasurementOptions:   metric.WithAttributeSet(costReductionCategoryAttributes),
	}
}
