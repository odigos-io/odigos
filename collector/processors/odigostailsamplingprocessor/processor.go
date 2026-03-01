package odigostailsamplingprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type tailSamplingProcessor struct {
	logger *zap.Logger
	config *Config
}

func (p *tailSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	// TODO: implement tail sampling logic
	// - Iterate over td (ResourceSpans -> ScopeSpans -> Spans)
	// - Apply sampling policy and drop or keep spans/traces
	return td, nil
}

func newTailSamplingProcessor(logger *zap.Logger, cfg *Config) *tailSamplingProcessor {
	return &tailSamplingProcessor{
		logger: logger,
		config: cfg,
	}
}
