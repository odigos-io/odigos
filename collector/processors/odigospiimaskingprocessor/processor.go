package odigospiimaskingprocessor

import (
	"context"
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/api/actions"
)

type piiMaskingProcessor struct {
	logger        *zap.Logger
	categories    []actions.PiiCategory
	customMaskers []*regexp.Regexp
}

func newPiiMaskingProcessor(set processor.Settings, cfg *Config) (*piiMaskingProcessor, error) {
	customMaskers, err := compileCustomMaskers(cfg)
	if err != nil {
		return nil, err
	}
	return &piiMaskingProcessor{
		logger:        set.Logger,
		categories:    append([]actions.PiiCategory(nil), cfg.PiiCategories...),
		customMaskers: customMaskers,
	}, nil
}

func compileCustomMaskers(cfg *Config) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(cfg.CustomFormatMaskings)+len(cfg.CustomRegexMaskings))

	for i, masking := range cfg.CustomFormatMaskings {
		re, err := buildFormatMaskingRegex(masking.LookupKey, masking.DataFormat)
		if err != nil {
			return nil, fmt.Errorf("customFormatMaskings[%d]: %w", i, err)
		}
		out = append(out, re)
	}

	for i, masking := range cfg.CustomRegexMaskings {
		re, err := regexp.Compile(masking.Regex)
		if err != nil {
			return nil, fmt.Errorf("customRegexMaskings[%d]: invalid regex: %w", i, err)
		}
		out = append(out, re)
	}

	return out, nil
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

	for _, category := range p.categories {
		masked, applied := maskCategory(category, result)
		if applied {
			result = masked
			changed = true
		}
	}

	for _, re := range p.customMaskers {
		masked, applied := maskCaptureGroups(re, result)
		if applied {
			result = masked
			changed = true
		}
	}

	return result, changed
}
