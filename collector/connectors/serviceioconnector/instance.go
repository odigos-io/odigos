package serviceioconnector

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

// collectorInstanceIDFromResource returns the collector's service.instance.id from the telemetry resource.
func collectorInstanceIDFromResource(resource pcommon.Resource) string {
	v, ok := resource.Attributes().Get(string(semconv.ServiceInstanceIDKey))
	if !ok {
		return ""
	}
	return v.Str()
}
