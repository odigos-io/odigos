package odigostailsamplingprocessor

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/costreduction"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/highlyrelevant"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category/noisy"
	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/internal/metadata"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/common/odigosattributes"
)

type tailSamplingProcessor struct {
	logger                *zap.Logger
	config                *Config
	odigosConfigExtension collector.OdigosConfigExtension

	noisyOperationsCategoryMeasurementOptions metric.MeasurementOption
	highlyRelevantCategoryMeasurementOptions  metric.MeasurementOption
	costReductionCategoryMeasurementOptions   metric.MeasurementOption

	telemetryBuilder *metadata.TelemetryBuilder
}

func (p *tailSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {

	if p.odigosConfigExtension == nil {
		p.logger.Error("odigos config extension is not set, skipping tail sampling")
		return td, nil // for auto generated tests, and not to crash in case it somehow happens
	}

	traceID, shouldProcess, spanCount, err := checkPrerequists(td)
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
	p.recordMetrics(ctx, consts.SamplingCategoryNoise, noisyOperationRes.RulesEvalResults, tracePercentage, p.noisyOperationsCategoryMeasurementOptions)
	if noisyOperationRes.DecidingRule != nil {
		noisyOperationRule := noisyOperationRes.DecidingRule
		percentageAtMost := category.GetPercentageOrDefault0(noisyOperationRule.PercentageAtMost)
		keepTrace := tracePercentage <= percentageAtMost

		p.recordCategoryMatchMetrics(ctx, p.noisyOperationsCategoryMeasurementOptions, keepTrace, spanCount)

		if keepTrace || p.config.DryRun {
			enrichSpansWithSamplingAttributes(td, consts.SamplingCategoryNoise, noisyOperationRule.Id, noisyOperationRule.Name, percentageAtMost, p.config.DryRun, keepTrace, p.config.SpanSamplingAttributes)
			return td, nil
		}
		return ptrace.NewTraces(), nil
	}

	highlyRelevantRes := highlyrelevant.Evaluate(td, p.odigosConfigExtension)
	p.recordMetrics(ctx, consts.SamplingCategoryHighlyRelevant, highlyRelevantRes.RulesEvalResults, tracePercentage, p.highlyRelevantCategoryMeasurementOptions)
	if highlyRelevantRes.DecidingRule != nil {
		highlyRelevantOperationRule := highlyRelevantRes.DecidingRule
		percentageAtLeast := category.GetPercentageOrDefault100(highlyRelevantOperationRule.PercentageAtLeast)
		keepTrace := tracePercentage <= percentageAtLeast

		p.recordCategoryMatchMetrics(ctx, p.highlyRelevantCategoryMeasurementOptions, keepTrace, spanCount)

		if keepTrace || p.config.DryRun {
			enrichSpansWithSamplingAttributes(td, consts.SamplingCategoryHighlyRelevant, highlyRelevantOperationRule.Id, highlyRelevantOperationRule.Name, percentageAtLeast, p.config.DryRun, keepTrace, p.config.SpanSamplingAttributes)
			return td, nil
		}
		return ptrace.NewTraces(), nil
	}

	costReductionRes := costreduction.Evaluate(td, p.odigosConfigExtension)
	p.recordMetrics(ctx, consts.SamplingCategoryCostReduction, costReductionRes.RulesEvalResults, tracePercentage, p.costReductionCategoryMeasurementOptions)
	if costReductionRes.DecidingRule != nil {
		costReductionRule := costReductionRes.DecidingRule
		percentageAtMost := costReductionRule.PercentageAtMost
		keepTrace := tracePercentage <= percentageAtMost

		p.recordCategoryMatchMetrics(ctx, p.costReductionCategoryMeasurementOptions, keepTrace, spanCount)

		if keepTrace || p.config.DryRun {
			enrichSpansWithSamplingAttributes(td, consts.SamplingCategoryCostReduction, costReductionRule.Id, costReductionRule.Name, percentageAtMost, p.config.DryRun, keepTrace, p.config.SpanSamplingAttributes)
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

	tailSamplingConfig, ok := p.getTailSamplingConfig(resource)
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
	// the extension name is validated as not nil in the config validate function
	// and can be nil in tests
	if p.config.OdigosConfigExtension != nil {
		ext, found := host.GetExtensions()[*p.config.OdigosConfigExtension]
		if !found || ext == nil {
			return fmt.Errorf("odigos config extension not found")
		}
		odigosConfigExtension, ok := ext.(collector.OdigosConfigExtension)
		if !ok {
			return fmt.Errorf("the collector extension instance %s is not a valid odigos config extension", *p.config.OdigosConfigExtension)
		}
		p.odigosConfigExtension = odigosConfigExtension
	}
	return nil
}

func (p *tailSamplingProcessor) getTailSamplingConfig(resource pcommon.Resource) (*commonapisampling.TailSamplingSourceConfig, bool) {
	collectorConfig, ok := p.odigosConfigExtension.GetFromResource(resource)
	if !ok {
		return nil, false
	}
	if collectorConfig.TailSampling == nil {
		return nil, false
	}
	return collectorConfig.TailSampling, true
}

func (p *tailSamplingProcessor) recordMetrics(ctx context.Context, category consts.SamplingCategory, evalResult category.CategoryRulesEvaluationResults, tracePercentage float64, categoryMeasurementOptions metric.MeasurementOption) {

	// record per rule metrics.
	for _, result := range evalResult {

		rulesAttrs := p.ruleMetricsAttributes(category, result)
		rulesMeasurementOptions := metric.WithAttributes(rulesAttrs...)

		p.telemetryBuilder.OdigosSamplingSpanCheckCount.Add(ctx, int64(result.SpanCheckedCount), rulesMeasurementOptions)
		p.telemetryBuilder.OdigosSamplingTraceCheckCount.Add(ctx, 1, rulesMeasurementOptions)

		if result.SpanMatchedCount > 0 {
			p.telemetryBuilder.OdigosSamplingSpanMatchCount.Add(ctx, int64(result.SpanMatchedCount), rulesMeasurementOptions)
			p.telemetryBuilder.OdigosSamplingTraceMatchCount.Add(ctx, 1, rulesMeasurementOptions)

			// for each rule, record if it is evaluated to drop or keep the trace.
			// even if disabled or in dry run mode, we can still monitor the decision.
			dropped := tracePercentage > result.RulePercentage
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

func (p *tailSamplingProcessor) ruleMetricsAttributes(category consts.SamplingCategory, result *category.RuleEvaluationResult) []attribute.KeyValue {

	rulesAttrs := []attribute.KeyValue{
		attribute.String(odigosattributes.SamplingCategory, string(category)),
		attribute.String(odigosattributes.SamplingRuleId, result.RuleId),
		attribute.String(odigosattributes.SamplingRuleName, result.RuleName),
	}

	// if rule was evaluated but disabled, add an attribute so it's visible in the metrics.
	if result.RuleDisabled {
		rulesAttrs = append(rulesAttrs, attribute.Bool(odigosattributes.SamplingRuleDisabled, true))
	}

	if p.config.DryRun {
		rulesAttrs = append(rulesAttrs, attribute.Bool(odigosattributes.SamplingDryRun, true))
	}

	return rulesAttrs
}

func categoryMetricsAttributes(category consts.SamplingCategory, dryRun bool) []attribute.KeyValue {
	categoryAttrs := []attribute.KeyValue{
		attribute.String(odigosattributes.SamplingCategory, string(category)),
	}
	if dryRun {
		categoryAttrs = append(categoryAttrs, attribute.Bool(odigosattributes.SamplingDryRun, true))
	}
	return categoryAttrs
}

func newTailSamplingProcessor(logger *zap.Logger, cfg *Config, set component.TelemetrySettings) *tailSamplingProcessor {
	telemetryBuilder, err := metadata.NewTelemetryBuilder(set)
	if err != nil {
		logger.Error("failed to create telemetry builder", zap.Error(err))
	}

	// compute once to to avoid creating new sets on each call.
	noisyOperationsCategoryAttributes := categoryMetricsAttributes(consts.SamplingCategoryNoise, cfg.DryRun)
	highlyRelevantCategoryAttributes := categoryMetricsAttributes(consts.SamplingCategoryHighlyRelevant, cfg.DryRun)
	costReductionCategoryAttributes := categoryMetricsAttributes(consts.SamplingCategoryCostReduction, cfg.DryRun)

	return &tailSamplingProcessor{
		logger:           logger,
		config:           cfg,
		telemetryBuilder: telemetryBuilder,
		noisyOperationsCategoryMeasurementOptions: metric.WithAttributes(noisyOperationsCategoryAttributes...),
		highlyRelevantCategoryMeasurementOptions:  metric.WithAttributes(highlyRelevantCategoryAttributes...),
		costReductionCategoryMeasurementOptions:   metric.WithAttributes(costReductionCategoryAttributes...),
	}
}
