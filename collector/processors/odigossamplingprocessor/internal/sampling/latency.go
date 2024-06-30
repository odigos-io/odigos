package sampling

import (
	"errors"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type TraceLatencyRule struct {
	Endpoint  string `mapstructure:"endpoint"`
	Threshold int    `mapstructure:"threshold"`
	Service   string `mapstructure:"service"`
}

func (tlr *TraceLatencyRule) Validate() error {
	switch {
	case tlr.Threshold <= 0:
		return errors.New("threshold must be a positive integer")
	case tlr.Service == "":
		return errors.New("service cannot be empty")
	case tlr.Endpoint == "":
		return errors.New("endpoint cannot be empty")
	case !strings.HasPrefix(tlr.Endpoint, "/"):
		return errors.New("endpoint must start with '/'")
	}
	return nil
}

func (tlr *TraceLatencyRule) TraceDropDecision(td ptrace.Traces) bool {
	var (
		serviceFound  bool
		endpointFound bool
	)

	resources := td.ResourceSpans()

	// Check if the service matches
	for r := 0; r < resources.Len(); r++ {
		serviceAttr, _ := resources.At(r).Resource().Attributes().Get(string(semconv.ServiceNameKey))
		if serviceAttr.AsString() == tlr.Service {
			serviceFound = true
		}
	}
	if !serviceFound {
		return false
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
					isEndpointFoundOnService := serviceName.AsString() == tlr.Service

					if tlr.matchEndpoint(endpoint.AsString(), tlr.Endpoint) && isEndpointFoundOnService {
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
	return endpointFound && duration.Milliseconds() < int64(tlr.Threshold)
}

func (tlr *TraceLatencyRule) matchEndpoint(rootSpanEndpoint string, samplerEndpoint string) bool {
	return strings.HasPrefix(rootSpanEndpoint, samplerEndpoint)
}
