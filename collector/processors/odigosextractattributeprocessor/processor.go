package odigosextractattributeprocessor

import (
	"context"
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

type extractAttributeProcessor struct {
	logger *zap.Logger
	config *Config
}

func newExtractAttributeProcessor(set processor.Settings, cfg *Config) *extractAttributeProcessor {
	return &extractAttributeProcessor{
		logger: set.Logger,
		config: cfg,
	}
}

func (p *extractAttributeProcessor) processTraces(ctx context.Context, traces ptrace.Traces) (ptrace.Traces, error) {
	allResourceSpans := traces.ResourceSpans()
	for i := 0; i < allResourceSpans.Len(); i++ {
		resourceSpans := allResourceSpans.At(i)
		allScopeSpans := resourceSpans.ScopeSpans()
		for j := 0; j < allScopeSpans.Len(); j++ {
			scopeSpans := allScopeSpans.At(j)
			spans := scopeSpans.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				p.processSpan(span)
			}
		}
	}
	return traces, nil
}

// Tries to match and get the attirbute from these two patterns via regex:
//  1. JSON-like fields: "key": "value", key:value, key = "value", etc.
//  2. URL path segments: /key/<value>.
//
// The first match gets returned
func extractAttributeFromJSONViaRegex(span ptrace.Span, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("empty extraction key")
	}

	jsonRe, urlRe := buildExtractionRegexes(key)

	var (
		result string
		found  bool
	)
	span.Attributes().Range(func(_ string, v pcommon.Value) bool {
		if v.Type() != pcommon.ValueTypeStr {
			return true
		}
		content := v.Str()
		if content == "" {
			return true
		}
		if m := jsonRe.FindStringSubmatch(content); len(m) > 1 {
			result = m[1]
			found = true
			return false
		}
		if m := urlRe.FindStringSubmatch(content); len(m) > 1 {
			result = m[1]
			found = true
			return false
		}
		return true
	})

	if !found {
		return "", fmt.Errorf("key %q not found in any span attribute", key)
	}
	return result, nil
}

func buildExtractionRegexes(key string) (*regexp.Regexp, *regexp.Regexp) {
	// Makes sure the key is escaped properly
	escaped_key := regexp.QuoteMeta(key)

	// Anchor the key on a boundary (start-of-string or a common JSON/SQL separator)
	// so substrings like "myfoo_bar" don't accidentally match "foo_bar".
	jsonRe := regexp.MustCompile(`(?:^|[\s,{("'])["']?` + escaped_key + `["']?\s*[:=]\s*["']?([^"'\s,;)}]+)`)
	urlRe := regexp.MustCompile(`/` + escaped_key + `/([^/\s"?&#]+)`)

	return jsonRe, urlRe
}

func addNewAttribute(span ptrace.Span, key string, value string) {
	span.Attributes().PutStr(key, value)
}

func (p *extractAttributeProcessor) processSpan(span ptrace.Span) {
	attributeValue, err := extractAttributeFromJSONViaRegex(span, p.config.SourceAttribute)
	if err != nil {
		p.logger.Debug("failed to extract attribute",
			zap.String("source_attribute", p.config.SourceAttribute),
			zap.String("target_attribute", p.config.TargetAttribute),
			zap.Stringer("spanId", span.SpanID()),
			zap.Error(err),
		)
		return
	}

	addNewAttribute(span, p.config.TargetAttribute, attributeValue)
}
