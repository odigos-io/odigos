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

func (tlr *HttpRouteLatencyRule) Validate() error {
	switch {
	case tlr.Threshold <= 0:
		return errors.New("threshold must be a positive integer")
	case tlr.ServiceName == "":
		return errors.New("service cannot be empty")
	case tlr.HttpRoute == "":
		return errors.New("endpoint cannot be empty")
	case !strings.HasPrefix(tlr.HttpRoute, "/"):
		return errors.New("endpoint must start with '/'")
	}
	return nil
}

func (tlr *HttpRouteLatencyRule) KeepTraceDecision(td ptrace.Traces) (filterMatch bool, conditionMatch bool) {
	var (
		serviceFound  = false
		endpointFound = false
	)

	resources := td.ResourceSpans()

	// Check if the service matches
	for r := 0; r < resources.Len(); r++ {
		serviceAttr, _ := resources.At(r).Resource().Attributes().Get(string(semconv.ServiceNameKey))
		if serviceAttr.AsString() == tlr.ServiceName {
			serviceFound = true
		}
	}
	if !serviceFound {
		return false, true
	}

	var minStart pcommon.Timestamp
	var maxEnd pcommon.Timestamp

	// Iterate over resources
	for r := 0; r < resources.Len(); r++ {
		scoreSpan := resources.At(r).ScopeSpans()

		// Iterate over scopes
		for j := 0; j < scoreSpan.Len(); j++ {
			ils := scoreSpan.At(j)

			// iterate over spans
			for k := 0; k < ils.Spans().Len(); k++ {
				span := ils.Spans().At(k)

				endpoint, found := span.Attributes().Get("http.route")
				if found {
					serviceName, _ := resources.At(r).Resource().Attributes().Get(string(semconv.ServiceNameKey))
					isEndpointFoundOnService := serviceName.AsString() == tlr.ServiceName

					if tlr.matchEndpoint(endpoint.AsString(), tlr.HttpRoute) && isEndpointFoundOnService {
						endpointFound = true
					}
				}

				start := span.StartTimestamp()
				end := span.EndTimestamp()

				if minStart == 0 || start < minStart {
					minStart = start
				}
				if maxEnd == 0 || end > maxEnd {
					maxEnd = end
				}
			}
		}
	}
	duration := maxEnd.AsTime().Sub(minStart.AsTime())
	return endpointFound, duration.Milliseconds() > int64(tlr.Threshold)
}

func (tlr *HttpRouteLatencyRule) matchEndpoint(rootSpanEndpoint string, samplerEndpoint string) bool {
	return strings.HasPrefix(rootSpanEndpoint, samplerEndpoint)
}
