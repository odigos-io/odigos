package sampling

import (
	"errors"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type HttpRouteLatencyRule struct {
	HttpRoute             string  `mapstructure:"http_route"`
	Threshold             int     `mapstructure:"threshold"`
	ServiceName           string  `mapstructure:"service_name"`
	FallbackSamplingRatio float64 `mapstructure:"fallback_sampling_ratio"`
}

var _ SamplingDecision = (*HttpRouteLatencyRule)(nil)

// Validate ensures the rule is well-formed and ready to evaluate.
func (r *HttpRouteLatencyRule) Validate() error {
	switch {
	case r.Threshold <= 0:
		return errors.New("threshold must be a positive integer")
	case r.ServiceName == "":
		return errors.New("service_name cannot be empty")
	case r.HttpRoute == "":
		return errors.New("http_route cannot be empty")
	case !strings.HasPrefix(r.HttpRoute, "/"):
		return errors.New("http_route must start with '/'")
	case r.FallbackSamplingRatio < 0 || r.FallbackSamplingRatio > 100:
		return errors.New("fallback_sampling_ratio must be between 0 and 100")
	}
	return nil
}

// Evaluate checks if the trace contains spans for the target service and HTTP route,
// and whether the trace latency exceeds the threshold. Sampling ratios are returned
// for the RuleEngine to apply.
// - matched: True if both endpoint and service name has matched.
// - satisfied: True if the latency was higher than the threshold
// - samplingRatio: sample ration on satisfy and fallback ration otherwise
func (r *HttpRouteLatencyRule) Evaluate(td ptrace.Traces) (bool, bool, float64) {
	resources := td.ResourceSpans()
	var serviceFound, endpointFound bool
	var minStart, maxEnd pcommon.Timestamp

	for i := 0; i < resources.Len(); i++ {
		resourceAttrs := resources.At(i).Resource().Attributes()
		serviceAttr, found := resourceAttrs.Get(string(semconv.ServiceNameKey))
		if !found || serviceAttr.AsString() != r.ServiceName {
			continue
		}
		serviceFound = true

		scopeSpans := resources.At(i).ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)

				if endpointAttr, ok := span.Attributes().Get("http.route"); ok {
					if r.matchEndpoint(endpointAttr.AsString(), r.HttpRoute) {
						endpointFound = true
					}
				}

				start, end := span.StartTimestamp(), span.EndTimestamp()
				if minStart == 0 || start < minStart {
					minStart = start
				}
				if maxEnd == 0 || end > maxEnd {
					maxEnd = end
				}
			}
		}
	}

	if !serviceFound || !endpointFound {
		// No match → rule doesn't apply
		return false, false, 0
	}

	// Compute total trace latency
	duration := maxEnd.AsTime().Sub(minStart.AsTime()).Milliseconds()

	if duration >= int64(r.Threshold) {
		// Latency condition satisfied → sample fully
		return true, true, 100.0
	}

	// Matched, but not satisfied → fallback applies
	return true, false, r.FallbackSamplingRatio
}

func (r *HttpRouteLatencyRule) matchEndpoint(spanEndpoint string, ruleEndpoint string) bool {
	// Match on prefix to allow hierarchical route matching
	return strings.HasPrefix(spanEndpoint, ruleEndpoint)
}
