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

var RELEVANT_SPAN_ATTRIBUTES = map[string]struct{}{
	"db.statement":              {},
	"db.query.text":             {},
	"messaging.message.payload": {},
	"http.request.payload":      {},
	"http.response.payload":     {},
}

type extractor struct {
	regex            *regexp.Regexp
	targetAttributeName string
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
			regex, err = buildExtractionRegex(extraction.LookupKey, extraction.DataFormat)
			if err != nil {
				return nil, fmt.Errorf("extractions[%d]: %w", i, err)
			}
		}
		out = append(out, extractor{regex: regex, targetAttributeName: extraction.TargetAttributeName})
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
		// Don't override an attribute that already exists on the span
		if _, exists := span.Attributes().Get(e.targetAttributeName); exists {
			continue
		}
		if value, ok := extractFromPayload(span, e.regex); ok {
			p.logger.Debug("extraction matched",
				zap.String("target_attribute_name", e.targetAttributeName),
				zap.String("value", value),
				zap.String("regex", e.regex.String()),
				zap.Stringer("spanId", span.SpanID()),
			)
			span.Attributes().PutStr(e.targetAttributeName, value)
		}
	}
}

// extractFromPayload scans the span's string-valued attributes which are payloads and returns the first capture group re produces.
func extractFromPayload(span ptrace.Span, re *regexp.Regexp) (string, bool) {

	var (
		result string
		found  bool
	)
	span.Attributes().Range(func(key string, value pcommon.Value) bool {
		// Check if the key is in our relevant attributes array
		if _, found := RELEVANT_SPAN_ATTRIBUTES[key]; !found {
			return true
		}
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
	case FormatJSON:
		// Examples (key = "user_id"):
		//   Quoted:     {"user_id": "abc123", "name": "foo"}   -> captures "abc123"
		//   Unquoted:   {user_id: 42, name: "foo"}             -> captures "42"
		//   Tight:      {"user_id":"abc"}                      -> captures "abc"
		//   Nested:     {"outer":{"user_id":"x"}}              -> captures "x"
		// Separator is ":" (JSON). Key must be preceded by start-of-string, whitespace, "{", or ",", optionally
		// wrapped in quotes, so substrings like "my_user_id" do NOT match.
		return regexp.MustCompile(`(?:^|[\s,{])"?` + escapedKey + `"?\s*:\s*"?([^"\s,}\]]+)`), nil
	case FormatSQL:
		// Examples (key = "user_id"):
		//   Quoted:     WHERE user_id = '42' AND status = 'ok'  -> captures "42"
		//   Tight:      WHERE user_id='abc'                     -> captures "abc"
		//   Unquoted:   WHERE user_id=42                        -> captures "42"
		//   Multiline:  "...\n      WHERE user_id = '42'\n..."  -> captures "42"
		// Separator is "=" (SQL). Key must be preceded by start-of-string, whitespace, "(", or ",", so substrings
		// like "my_user_id" do NOT match.
		return regexp.MustCompile(`(?:^|[\s,(])` + escapedKey + `\s*=\s*'?([^'\s,;)]+)`), nil
	case FormatResourcePath:
		// Examples (key = "orders"):
		//   Path:           /api/v1/orders/abc-123                     -> captures "abc-123"
		//   Full URL:       https://example.com/orders/42?foo=bar      -> captures "42"
		//   Relative:       orders/42/items                            -> captures "42"
		// Stops at the next "/", whitespace, "?", "&", "#", or quote so query strings and fragments are excluded.
		return regexp.MustCompile(`(?:^|/)` + escapedKey + `/([^/\s"?&#]+)`), nil
	default:
		return nil, fmt.Errorf("unsupported data_format %q", format)
	}
}
