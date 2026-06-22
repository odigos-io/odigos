package serviceioconnector

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"

	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/connectors/serviceioconnector/internal/metadata"
)

const (
	metricNameConnectionTotal = "traces_service_io_connection_total"
	serviceNameAttribute      = string(semconv.ServiceNameKey)
	inputAttributePrefix      = "input."
	outputAttributePrefix     = "output."
	// Identifies the collector pod/process that produced the metric (distinct from workload k8s.pod.name on spans).
	collectorInstanceAttributeId = "odigos.collector.instance.id"
)

var metricResourceAttributeKeys = ServiceInstanceRuntimeAttributeKeys

func buildServiceInstanceBaseAttributes(instance *ServiceInstance, inputAttrs pcommon.Map) pcommon.Map {
	attributes := pcommon.NewMap()
	if instance.ServiceName != "" {
		attributes.PutStr(serviceNameAttribute, instance.ServiceName)
	}
	mergeAttributes(instance.ResourceAttributes, attributes)
	mergeAttributes(inputAttrs, attributes)
	return attributes
}

func buildMetricResourceAttributes(instance *ServiceInstance) pcommon.Map {
	resource := pcommon.NewMap()
	for _, key := range metricResourceAttributeKeys {
		copyStringAttributeIfPresent(instance.ResourceAttributes, resource, key)
	}
	return resource
}

func copyStringAttributeIfPresent(source, destination pcommon.Map, key string) {
	value, ok := source.Get(key)
	if !ok || value.Type() != pcommon.ValueTypeStr || value.Str() == "" {
		return
	}
	destination.PutStr(key, value.Str())
}

func buildConnectionAttributes(inputAnrResourceAttributes, outputAttrs pcommon.Map) (uint64, pcommon.Map) {
	attributes := pcommon.NewMap()
	inputAnrResourceAttributes.CopyTo(attributes)
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

func (c *serviceioConnector) aggregateConnectionsFromTree(tree *TraceTree) bool {
	now := time.Now()

	c.seriesMutex.Lock()
	defer c.seriesMutex.Unlock()

	c.pruneStaleSeriesLocked(now)

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
				if c.maxMetricSeries > 0 && len(c.keyToMetric) >= c.maxMetricSeries {
					c.seriesLimitOnce.Do(func() {
						if c.logger != nil {
							c.logger.Warn(
								"serviceio connector metric series limit reached; dropping new connection series",
								zap.Int("max_metric_series", c.maxMetricSeries),
							)
						}
					})
					continue
				}
				series.dimensions = attributes
				series.resource = buildMetricResourceAttributes(instance)
			}
			series.count++
			series.updatedAt = now
			c.keyToMetric[key] = series
			added = true
		}
	}

	return added
}

func metricScopeName() string {
	return metadata.ScopeName
}
