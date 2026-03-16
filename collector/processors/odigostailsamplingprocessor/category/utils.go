package category

import (
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func GetPercentageOrDefault(percentage *float64, defaultValue float64) float64 {
	if percentage == nil {
		return defaultValue
	}
	return *percentage
}

func GetPercentageOrDefault100(percentage *float64) float64 {
	return GetPercentageOrDefault(percentage, 100.0)
}

// SpanDurationNano returns the span duration in nanoseconds (EndTimestamp - StartTimestamp).
func SpanDurationNano(span ptrace.Span) uint64 {
	return uint64(span.EndTimestamp() - span.StartTimestamp())
}
