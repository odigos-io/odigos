package serviceioconnector

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestBuildMetricResourceAttributes(t *testing.T) {
	resourceAttributes := pcommon.NewMap()
	resourceAttributes.PutStr(TelemetrySDKLanguageAttribute, "java")
	resourceAttributes.PutStr(ProcessRuntimeNameAttribute, "OpenJDK Runtime Environment")
	resourceAttributes.PutStr(ProcessRuntimeVersionAttribute, "17.0.12")
	resourceAttributes.PutStr("service.name", "checkout")

	instance := &ServiceInstance{
		ResourceAttributes: resourceAttributes,
	}

	resource := buildMetricResourceAttributes(instance)
	language, ok := resource.Get(TelemetrySDKLanguageAttribute)
	require.True(t, ok)
	require.Equal(t, "java", language.Str())

	runtimeName, ok := resource.Get(ProcessRuntimeNameAttribute)
	require.True(t, ok)
	require.Equal(t, "OpenJDK Runtime Environment", runtimeName.Str())

	runtimeVersion, ok := resource.Get(ProcessRuntimeVersionAttribute)
	require.True(t, ok)
	require.Equal(t, "17.0.12", runtimeVersion.Str())

	_, ok = resource.Get("service.name")
	require.False(t, ok)
}

func TestBuildMetricsSetsResourceAttributes(t *testing.T) {
	connector := &serviceioConnector{
		config:              &Config{},
		startTime:           pcommon.Timestamp(1).AsTime(),
		keyToMetric:         make(map[uint64]metricSeries),
		collectorInstanceID: "collector-a",
	}

	resourceAttributes := pcommon.NewMap()
	resourceAttributes.PutStr(TelemetrySDKLanguageAttribute, "python")
	resourceAttributes.PutStr(ProcessRuntimeNameAttribute, "CPython")
	resourceAttributes.PutStr(ProcessRuntimeVersionAttribute, "3.12.1")

	instance := &ServiceInstance{
		ServiceName:        "orders",
		ResourceAttributes: resourceAttributes,
	}
	inputAttrs := pcommon.NewMap()
	inputAttrs.PutStr(inputAttributePrefix+"http.route", "/orders")
	outputAttrs := pcommon.NewMap()
	outputAttrs.PutStr(outputAttributePrefix+"db.system", "postgresql")

	inputAttributes := buildServiceInstanceBaseAttributes(instance, inputAttrs)
	key, attributes := buildConnectionAttributes(inputAttributes, outputAttrs)
	connector.keyToMetric[key] = metricSeries{
		dimensions: attributes,
		resource:   buildMetricResourceAttributes(instance),
		count:      2,
	}

	md, err := connector.buildMetrics()
	require.NoError(t, err)
	require.Equal(t, 1, md.ResourceMetrics().Len())

	resource := md.ResourceMetrics().At(0).Resource().Attributes()
	language, ok := resource.Get(TelemetrySDKLanguageAttribute)
	require.True(t, ok)
	require.Equal(t, "python", language.Str())
}
