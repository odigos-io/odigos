package odigosconditionalattributes

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	OTTLScopeNameKey = "instrumentation_scope.name"
)

type conditionalAttributesProcessor struct {
	logger              *zap.Logger
	config              *Config
	uniqueNewAttributes map[string]struct{}
}

// ============================================================================
// Traces Processing
// ============================================================================

func (p *conditionalAttributesProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		resourceSpans := rss.At(i)
		ilss := resourceSpans.ScopeSpans()
		for j := 0; j < ilss.Len(); j++ {
			scopeSpans := ilss.At(j)
			spans := scopeSpans.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				p.processSpan(span, scopeSpans.Scope().Name(), scopeSpans.Scope().Attributes(), resourceSpans.Resource().Attributes())
			}
		}
	}
	return td, nil
}

func (p *conditionalAttributesProcessor) processSpan(span ptrace.Span, scopeName string, scopeAttributes pcommon.Map, resourceAttributes pcommon.Map) {
	// Retrieve span attributes.
	attributes := span.Attributes()

	// Iterate over each rule in the configuration.
	for _, rule := range p.config.Rules {
		// Add attributes based on the rule (handles missing AttributeToCheck internally).
		p.addAttributes(attributes, rule, scopeName, scopeAttributes, resourceAttributes)
	}
	p.setDefaultValueAttributes(attributes)
}

func (p *conditionalAttributesProcessor) addAttributes(spanAttributes pcommon.Map, rule ConditionalRule,
	scopeName string, scopeAttributes pcommon.Map, resourceAttributes pcommon.Map) {

	// Handle cases where rule checks for scope_name ['instrumentation_scope.name'].
	if rule.FieldToCheck == OTTLScopeNameKey {
		p.handleScopeNameConditionalAttribute(spanAttributes, rule, scopeName)
		return
	}

	var attrStr string

	attributeSets := []pcommon.Map{spanAttributes, scopeAttributes, resourceAttributes}
	for _, attrs := range attributeSets {
		if attrValue, ok := attrs.Get(rule.FieldToCheck); ok {
			attrStr = attrValue.AsString()
			break
		}
	}

	// Check if the value matches a configured value in the rule.
	if valueConfig, exists := rule.NewAttributeValueConfigurations[attrStr]; exists {
		for _, configAction := range valueConfig {

			// Add a static value as a new attribute if defined.
			if configAction.Value != "" {
				if _, exists := spanAttributes.Get(configAction.NewAttributeName); !exists {
					spanAttributes.PutStr(configAction.NewAttributeName, configAction.Value)
				} else if _, exists := scopeAttributes.Get(configAction.NewAttributeName); !exists {
					spanAttributes.PutStr(configAction.NewAttributeName, configAction.Value)
				} else if _, exists := resourceAttributes.Get(configAction.NewAttributeName); !exists {
					spanAttributes.PutStr(configAction.NewAttributeName, configAction.Value)
				}
			} else if configAction.FromField != "" { // Copy a value from another attribute if specified.
				if fromAttrValue, ok := spanAttributes.Get(configAction.FromField); ok {
					if _, exists := spanAttributes.Get(configAction.NewAttributeName); !exists {
						spanAttributes.PutStr(configAction.NewAttributeName, fromAttrValue.AsString())
					} else if _, exists := scopeAttributes.Get(configAction.NewAttributeName); !exists {
						spanAttributes.PutStr(configAction.NewAttributeName, fromAttrValue.AsString())
					} else if _, exists := resourceAttributes.Get(configAction.NewAttributeName); !exists {
						spanAttributes.PutStr(configAction.NewAttributeName, fromAttrValue.AsString())
					}
				}
			}

		}

	}
}

func (p *conditionalAttributesProcessor) handleScopeNameConditionalAttribute(
	spanAttributes pcommon.Map,
	rule ConditionalRule,
	scopeName string,
) {
	if valueConfigActions, exists := rule.NewAttributeValueConfigurations[scopeName]; exists {
		for _, configAction := range valueConfigActions {
			if configAction.Value != "" {
				// Add static value if not already present
				if _, exists := spanAttributes.Get(configAction.NewAttributeName); !exists {
					spanAttributes.PutStr(configAction.NewAttributeName, configAction.Value)
				}
			} else if configAction.FromField != "" {
				// Copy value from another attribute if defined
				if fromAttrValue, ok := spanAttributes.Get(configAction.FromField); ok {
					if _, exists := spanAttributes.Get(configAction.NewAttributeName); !exists {
						spanAttributes.PutStr(configAction.NewAttributeName, fromAttrValue.AsString())
					}
				}
			}
		}
	}
}

// Set default values for unique attributes if not already set
func (p *conditionalAttributesProcessor) setDefaultValueAttributes(
	spanAttributes pcommon.Map,
) {
	for uniqueAttribute := range p.uniqueNewAttributes {
		if _, exists := spanAttributes.Get(uniqueAttribute); !exists {
			spanAttributes.PutStr(uniqueAttribute, p.config.GlobalDefault)
		}
	}
}

// ============================================================================
// Metrics Processing
// ============================================================================

func (p *conditionalAttributesProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		resourceMetrics := rms.At(i)
		ilms := resourceMetrics.ScopeMetrics()
		for j := 0; j < ilms.Len(); j++ {
			scopeMetrics := ilms.At(j)
			metrics := scopeMetrics.Metrics()
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				p.processMetric(metric, resourceMetrics.Resource().Attributes())
			}
		}
	}
	return md, nil
}

func (p *conditionalAttributesProcessor) processMetric(metric pmetric.Metric, resourceAttributes pcommon.Map) {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dps := metric.Gauge().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			p.processMetricDataPoint(dps.At(i).Attributes(), resourceAttributes)
		}
	case pmetric.MetricTypeSum:
		dps := metric.Sum().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			p.processMetricDataPoint(dps.At(i).Attributes(), resourceAttributes)
		}
	case pmetric.MetricTypeHistogram:
		dps := metric.Histogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			p.processMetricDataPoint(dps.At(i).Attributes(), resourceAttributes)
		}
	case pmetric.MetricTypeExponentialHistogram:
		dps := metric.ExponentialHistogram().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			p.processMetricDataPoint(dps.At(i).Attributes(), resourceAttributes)
		}
	case pmetric.MetricTypeSummary:
		dps := metric.Summary().DataPoints()
		for i := 0; i < dps.Len(); i++ {
			p.processMetricDataPoint(dps.At(i).Attributes(), resourceAttributes)
		}
	}
}

func (p *conditionalAttributesProcessor) processMetricDataPoint(dataPointAttributes pcommon.Map, resourceAttributes pcommon.Map) {
	// Iterate over each rule in the configuration.
	for _, rule := range p.config.Rules {
		// Skip rules that don't have field_to_check_metrics defined
		if rule.FieldToCheckMetrics == "" {
			continue
		}
		// Add attributes based on the rule
		p.addAttributesForMetrics(dataPointAttributes, rule, resourceAttributes)
	}
	p.setDefaultValueAttributes(dataPointAttributes)
}

func (p *conditionalAttributesProcessor) addAttributesForMetrics(dataPointAttributes pcommon.Map, rule ConditionalRule,
	resourceAttributes pcommon.Map) {

	var attrStr string

	attributeSets := []pcommon.Map{dataPointAttributes, resourceAttributes}
	for _, attrs := range attributeSets {
		if attrValue, ok := attrs.Get(rule.FieldToCheckMetrics); ok {
			attrStr = attrValue.AsString()
			break
		}
	}

	if attrStr == "" {
		return
	}

	// Check if the value matches a configured value in the rule.
	if valueConfig, exists := rule.NewAttributeValueConfigurations[attrStr]; exists {
		for _, configAction := range valueConfig {

			// Add a static value as a new attribute if defined.
			if configAction.Value != "" {
				if _, exists := dataPointAttributes.Get(configAction.NewAttributeName); !exists {
					dataPointAttributes.PutStr(configAction.NewAttributeName, configAction.Value)
				} else if _, exists := resourceAttributes.Get(configAction.NewAttributeName); !exists {
					dataPointAttributes.PutStr(configAction.NewAttributeName, configAction.Value)
				}
			} else if configAction.FromField != "" { // Copy a value from another attribute if specified.
				if fromAttrValue, ok := dataPointAttributes.Get(configAction.FromField); ok {
					if _, exists := dataPointAttributes.Get(configAction.NewAttributeName); !exists {
						dataPointAttributes.PutStr(configAction.NewAttributeName, fromAttrValue.AsString())
					} else if _, exists := resourceAttributes.Get(configAction.NewAttributeName); !exists {
						dataPointAttributes.PutStr(configAction.NewAttributeName, fromAttrValue.AsString())
					}
				}
			}

		}

	}
}
