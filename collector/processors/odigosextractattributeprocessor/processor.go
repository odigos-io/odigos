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

type extractor struct {
	regex  *regexp.Regexp
	target string
}

type extractAttributeProcessor struct {
	logger     *zap.Logger
	extractors []extractor
}

func newExtractAttributeProcessor(set processor.Settings, cfg *Config) (*extractAttributeProcessor, error) {
	// Compile the regex extractors at init so we don't calculate them when processing each span
	extractors, err := compileRegexExtractors(cfg)
	if err != nil {
		return nil, err
	}
	return &extractAttributeProcessor{
		logger:     set.Logger,
		extractors: extractors,
	}, nil
}

// compileRegexExtractors precompiles one regex per Extraction entry at startup so the per-span path stays allocation-free.
func compileRegexExtractors(cfg *Config) ([]extractor, error) {
	out := make([]extractor, 0, len(cfg.Extractions))
	for i, extraction := range cfg.Extractions {
		var regex *regexp.Regexp
		var err error

		if extraction.Regex != "" {
			regex, err = regexp.Compile(extraction.Regex)
			if err != nil {
				return nil, fmt.Errorf("extractions[%d]: invalid regex: %w", i, err)
			}
		} else {
			regex, err = buildExtractionRegex(extraction.Source, extraction.DataFormat)
			if err != nil {
				return nil, fmt.Errorf("extractions[%d]: %w", i, err)
			}
		}
		out = append(out, extractor{regex: regex, target: extraction.Target})
	}
	return out, nil
}

func (p *extractAttributeProcessor) processTraces(_ context.Context, traces ptrace.Traces) (ptrace.Traces, error) {
	allResourceSpans := traces.ResourceSpans()
	for i := 0; i < allResourceSpans.Len(); i++ {
		allScopeSpans := allResourceSpans.At(i).ScopeSpans()
		for j := 0; j < allScopeSpans.Len(); j++ {
			spans := allScopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				p.processSpan(spans.At(k))
			}
		}
	}
	return traces, nil
}

func (p *extractAttributeProcessor) processSpan(span ptrace.Span) {
	for _, e := range p.extractors {
		if value, ok := extractFromAttributes(span, e.regex); ok {
			p.logger.Debug("extraction matched",
				zap.String("target", e.target),
				zap.String("value", value),
				zap.String("regex", e.regex.String()),
				zap.Stringer("spanId", span.SpanID()),
			)
			span.Attributes().PutStr(e.target, value)
		} else {
			p.logger.Debug("extraction did not match any attribute",
				zap.String("target", e.target),
				zap.String("regex", e.regex.String()),
				zap.Stringer("spanId", span.SpanID()),
			)
		}
	}
}

// extractFromAttributes scans the span's string-valued attributes and returns the first capture group re produces.
func extractFromAttributes(span ptrace.Span, re *regexp.Regexp) (string, bool) {
	var (
		result string
		found  bool
	)
	span.Attributes().Range(func(_ string, value pcommon.Value) bool {
		if value.Type() != pcommon.ValueTypeStr {
			return true
		}
		content := value.Str()
		if content == "" {
			return true
		}
		// Take the first regex match
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			result = matches[1]
			found = true
			return false
		}
		return true
	})
	return result, found
}

// buildExtractionRegex returns the pattern that captures the value of key for the given format. The key is anchored
// on a JSON/SQL/URL boundary so substrings like "myfoo_bar" don't cross-match "foo_bar".
func buildExtractionRegex(key string, format DataFormat) (*regexp.Regexp, error) {
	escapedKey := regexp.QuoteMeta(key)
	switch format {
	case FormatJSON: // Also works for SQL
		return regexp.MustCompile(`(?:^|[\s,{("'])["']?` + escapedKey + `["']?\s*[:=]\s*["']?([^"'\s,;)}]+)`), nil
	case FormatURL:
		return regexp.MustCompile(`(?:^|/)` + escapedKey + `/([^/\s"?&#]+)`), nil
	default:
		return nil, fmt.Errorf("unsupported data_format %q", format)
	}
}
