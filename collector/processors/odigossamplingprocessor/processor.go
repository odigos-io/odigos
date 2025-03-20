package odigossamplingprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type samplingProcessor struct {
	logger *zap.Logger
	config *Config
	engine *RuleEngine
}

func (sp *samplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if !sp.engine.ShouldSample(td) {
		sp.removeAllSpans(&td)
	}
	return td, nil
}

func (sp *samplingProcessor) removeAllSpans(td *ptrace.Traces) {
	td.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool { return true })
}

func newSamplingProcessor(logger *zap.Logger, cfg *Config) *samplingProcessor {
	return &samplingProcessor{
		logger: logger,
		config: cfg,
		engine: NewRuleEngine(cfg, logger),
	}
}
