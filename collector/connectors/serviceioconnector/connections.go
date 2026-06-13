package serviceioconnector

import (
	"go.opentelemetry.io/collector/pdata/pcommon"

	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	"github.com/odigos-io/odigos/collector/connectors/serviceioconnector/internal/metadata"
	"github.com/odigos-io/odigos/collector/pkg/completetrace"
)

const (
	metricNameConnectionTotal = "traces_service_io_connection_total"
	serviceNameAttribute      = string(semconv.ServiceNameKey)
	inputAttributePrefix      = "input."
	outputAttributePrefix     = "output."
)

func buildServiceInstanceBaseAttributes(instance *completetrace.ServiceInstance, inputAttrs pcommon.Map) pcommon.Map {
	attributes := pcommon.NewMap()
	if instance.ServiceName != "" {
		attributes.PutStr(serviceNameAttribute, instance.ServiceName)
	}
	mergeAttributes(instance.ResourceAttributes, attributes)
	mergeAttributes(inputAttrs, attributes)
	return attributes
}

func buildConnectionAttributes(inputAttributes, outputAttrs pcommon.Map) (uint64, pcommon.Map) {
	attributes := pcommon.NewMap()
	inputAttributes.CopyTo(attributes)
	mergeAttributes(outputAttrs, attributes)
	return hashAttributes(attributes), attributes
}

func mergeAttributes(source, destination pcommon.Map) {
	if source == (pcommon.Map{}) {
		return
	}
	source.Range(func(name string, value pcommon.Value) bool {
		value.CopyTo(destination.PutEmpty(name))
		return true
	})
}

func (c *serviceioConnector) aggregateConnectionsFromTree(tree *completetrace.TraceTree) bool {
	c.seriesMutex.Lock()
	defer c.seriesMutex.Unlock()

	added := false
	for _, instance := range tree.ServiceInstances {
		if !c.isActiveSourceInstance(instance) {
			continue
		}

		inputAttrs := ExtractSpanAttributes(instance.Root, inputAttributePrefix, c.inputSpanAttributes)
		serviceInputBaseAttributes := buildServiceInstanceBaseAttributes(instance, inputAttrs)

		for _, outputLeaf := range instance.OutputLeaves {
			outputAttrs := ExtractSpanAttributes(outputLeaf, outputAttributePrefix, c.outputSpanAttributes)
			key, attributes := buildConnectionAttributes(serviceInputBaseAttributes, outputAttrs)
			series := c.keyToMetric[key]
			if series.count == 0 {
				series.dimensions = attributes
			}
			series.count++
			c.keyToMetric[key] = series
			added = true
		}
	}

	return added
}

func metricScopeName() string {
	return metadata.ScopeName
}
