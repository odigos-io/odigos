package odigostracefilterprocessor

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// w3cSampledBit is the W3C trace-flags sampled bit (bit 0).
// See https://www.w3.org/TR/trace-context/#trace-flags
const (
	w3cSampledBit uint32 = 0x01

	telemetrySDKLanguageAttributeName = "telemetry.sdk.language"
	rubyTelemetrySDKLanguage          = "ruby"
)

// SpanFilterEvaluator determines whether a span should be kept or dropped.
// Returns true if the span should be dropped.
type SpanFilterEvaluator interface {
	ShouldDrop(resource pcommon.Resource, span ptrace.Span) bool
}

// unsampledBitEvaluator drops spans where the W3C sampled bit (bit 0) is not set.
//
// Some OTLP exporters do not populate the W3C trace flags (bits 0-7) in Span.Flags,
// only setting the parent-isRemote bits (8-9). For these SDKs the sampled bit can
// be 0 even for sampled spans.
// Known affected: Ruby SDK.
// See: https://github.com/open-telemetry/opentelemetry-ruby/issues/1917
type unsampledBitEvaluator struct{}

func (e *unsampledBitEvaluator) ShouldDrop(resource pcommon.Resource, span ptrace.Span) bool {
	if hasRubySDKLanguage(resource) {
		return false
	}

	return span.Flags()&w3cSampledBit == 0
}

func hasRubySDKLanguage(resource pcommon.Resource) bool {
	language, ok := resource.Attributes().Get(telemetrySDKLanguageAttributeName)
	return ok && language.AsString() == rubyTelemetrySDKLanguage
}
