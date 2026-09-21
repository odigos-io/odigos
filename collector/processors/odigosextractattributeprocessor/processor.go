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
	// regexes are tried in order and the first capture produced wins, so a preset
	// format can prefer its quoted-value pattern over the unquoted one.
	regexes             []*regexp.Regexp
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

// compileRegexExtractors precompiles the regexes of every Extraction entry at startup so the per-span path stays allocation-free.
func compileRegexExtractors(cfg *Config) ([]extractor, error) {
	out := make([]extractor, 0, len(cfg.Extractions))
	for i, extraction := range cfg.Extractions {
		var regexes []*regexp.Regexp

		if extraction.Regex != "" {
			regex, err := regexp.Compile(extraction.Regex)
			if err != nil {
				return nil, fmt.Errorf("extractions[%d]: invalid regex: %w", i, err)
			}
			regexes = []*regexp.Regexp{regex}
		} else {
			built, err := buildExtractionRegexes(extraction.LookupKey, extraction.DataFormat)
			if err != nil {
				return nil, fmt.Errorf("extractions[%d]: %w", i, err)
			}
			regexes = built
		}
		out = append(out, extractor{regexes: regexes, targetAttributeName: extraction.TargetAttributeName})
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
		if value, matched, ok := extractFromPayload(span, e.regexes); ok {
			p.logger.Debug("extraction matched",
				zap.String("target_attribute_name", e.targetAttributeName),
				zap.String("value", value),
				zap.String("regex", matched.String()),
				zap.Stringer("spanId", span.SpanID()),
			)
			span.Attributes().PutStr(e.targetAttributeName, value)
		}
	}
}

// extractFromPayload scans the span's string-valued attributes which are payloads and returns the first capture group
// the regexes produce, along with the regex that produced it.
func extractFromPayload(span ptrace.Span, regexes []*regexp.Regexp) (string, *regexp.Regexp, bool) {

	var (
		result  string
		matched *regexp.Regexp
		found   bool
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
		for _, re := range regexes {
			if matches := re.FindStringSubmatch(content); len(matches) > 1 {
				result = matches[1]
				matched = re
				found = true
				return false
			}
		}
		return true
	})
	return result, matched, found
}

// buildExtractionRegexes returns the patterns that capture the value of key for the given format. The key is anchored
// on a JSON/SQL/URL boundary so substrings like "myfoo_bar" don't cross-match "foo_bar".
//
// JSON and SQL get two patterns: a quoted one that captures the whole value up to the closing quote
// (values may contain spaces and commas, e.g. a full name or a street address), and the unquoted one
// for values that are not wrapped in quotes. The patterns are tried in order, so a quoted value is
// captured in full and the unquoted pattern only applies when the value carries no quotes.
// This mirrors buildFormatMaskingRegexes in odigospiimaskingprocessor.
func buildExtractionRegexes(key string, format DataFormat) ([]*regexp.Regexp, error) {
	escapedKey := regexp.QuoteMeta(key)
	switch format {
	case FormatJSON:
		// Examples (key = "user_id"):
		//   Quoted:     {"user_id": "abc123", "name": "foo"}   -> captures "abc123"
		//   With space: {"user_id": "abc 123"}                 -> captures "abc 123"
		//   Unquoted:   {user_id: 42, name: "foo"}             -> captures "42"
		//   Tight:      {"user_id":"abc"}                      -> captures "abc"
		//   Nested:     {"outer":{"user_id":"x"}}              -> captures "x"
		// Separator is ":" (JSON). Key must be preceded by start-of-string, whitespace, "{", or ",", optionally
		// wrapped in quotes, so substrings like "my_user_id" do NOT match.
		return compileAll(
			`(?:^|[\s,{])"?`+escapedKey+`"?\s*:\s*"((?:[^"\\]|\\.)+)"?`,
			`(?:^|[\s,{])"?`+escapedKey+`"?\s*:\s*"?([^"\s,}\]]+)`,
		)
	case FormatSQL:
		// Examples (key = "user_id"):
		//   Quoted:     WHERE user_id = '42' AND status = 'ok'  -> captures "42"
		//   With space: WHERE user_id = 'Jane Public'           -> captures "Jane Public"
		//   Tight:      WHERE user_id='abc'                     -> captures "abc"
		//   Unquoted:   WHERE user_id=42                        -> captures "42"
		//   Multiline:  "...\n      WHERE user_id = '42'\n..."  -> captures "42"
		// Separator is "=" (SQL). Key must be preceded by start-of-string, whitespace, "(", or ",", so substrings
		// like "my_user_id" do NOT match.
		return compileAll(
			`(?:^|[\s,(])`+escapedKey+`\s*=\s*'((?:[^'\\]|\\.)+)'?`,
			`(?:^|[\s,(])`+escapedKey+`\s*=\s*'?([^'\s,;)]+)`,
		)
	case FormatResourcePath:
		// Examples (key = "orders"):
		//   Path:           /api/v1/orders/abc-123                     -> captures "abc-123"
		//   Full URL:       https://example.com/orders/42?foo=bar      -> captures "42"
		//   Relative:       orders/42/items                            -> captures "42"
		// Stops at the next "/", whitespace, "?", "&", "#", or quote so query strings and fragments are excluded.
		return compileAll(`(?:^|/)` + escapedKey + `/([^/\s"?&#]+)`)
	default:
		return nil, fmt.Errorf("unsupported data_format %q", format)
	}
}

func compileAll(patterns ...string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}
