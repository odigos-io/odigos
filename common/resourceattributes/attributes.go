package resourceattributes

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Attributes []attribute.KeyValue

// ToEnvVarString converts the attributes to a string that can be used as OTEL_RESOURCE_ATTRIBUTES env var
func (a Attributes) ToEnvVarString() string {
	var attrs []string
	for _, attr := range a {
		attrs = append(attrs, fmt.Sprintf("%s=%s", attr.Key, attr.Value.AsString()))
	}
	return strings.Join(attrs, ",")
}

// IncludeServiceName adds the service name to the attributes
// OpAMP clients expect the service name to be set in the resource attributes
func (a Attributes) IncludeServiceName(serviceName string) Attributes {
	return append(a, semconv.ServiceName(serviceName))
}
