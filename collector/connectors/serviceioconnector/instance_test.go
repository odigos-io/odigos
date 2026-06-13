package serviceioconnector

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func TestCollectorInstanceIDFromResource(t *testing.T) {
	resource := pcommon.NewResource()
	resource.Attributes().PutStr(string(semconv.ServiceInstanceIDKey), "0a732fb6-8ee3-4e02-9d87-fb5025f829a6")

	id := collectorInstanceIDFromResource(component.TelemetrySettings{Resource: resource})
	require.Equal(t, "0a732fb6-8ee3-4e02-9d87-fb5025f829a6", id)
}

func TestCollectorInstanceIDFromResource_UnknownWhenMissing(t *testing.T) {
	id := collectorInstanceIDFromResource(component.TelemetrySettings{Resource: pcommon.NewResource()})
	require.Equal(t, "unknown", id)
}
