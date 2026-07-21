package odigospiimaskingprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type piiMaskingProcessor struct {
	logger *zap.Logger
	config *Config
}

func newPiiMaskingProcessor(set processor.Settings, cfg *Config) *piiMaskingProcessor {
	return &piiMaskingProcessor{
		logger: set.Logger,
		config: cfg,
	}
}

func (p *piiMaskingProcessor) processTraces(_ context.Context, traces ptrace.Traces) (ptrace.Traces, error) {
	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		scopeSpans := resourceSpans.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				p.processSpan(spans.At(k))
			}
		}
	}
	return traces, nil
}

func (p *piiMaskingProcessor) processSpan(span ptrace.Span) {
	span.Attributes().Range(func(_ string, value pcommon.Value) bool {
		p.processAttributeValue(value)
		return true
	})
}

func (p *piiMaskingProcessor) processAttributeValue(value pcommon.Value) {
	switch value.Type() {
	case pcommon.ValueTypeStr:
		if masked, changed := p.maskPiiData(value.Str()); changed {
			value.SetStr(masked)
		}
	case pcommon.ValueTypeSlice:
		slice := value.Slice()
		for i := 0; i < slice.Len(); i++ {
			p.processAttributeValue(slice.At(i))
		}
	}
}

func (p *piiMaskingProcessor) maskPiiData(value string) (string, bool) {
	result := value
	changed := false
	for _, category := range p.config.PiiCategories {
		masked, applied := maskCategory(category, result)
		if applied {
			result = masked
			changed = true
		}
	}
	return result, changed
}
