package odigosconditionalattributes

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
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
	if rule.AttributeToCheck == OTTLScopeNameKey {
		p.handleScopeNameConditionalAttribute(spanAttributes, rule, scopeName)
		return
	}

	var attrStr string

	attributeSets := []pcommon.Map{spanAttributes, scopeAttributes, resourceAttributes}
	for _, attrs := range attributeSets {
		if attrValue, ok := attrs.Get(rule.AttributeToCheck); ok {
			attrStr = attrValue.AsString()
			break
		}
	}

	// Check if the value matches a configured value in the rule.
	if valueConfig, exists := rule.Values[attrStr]; exists {
		for _, configAction := range valueConfig {

			// Add a static value as a new attribute if defined.
			if configAction.Value != "" {
				if _, exists := spanAttributes.Get(configAction.NewAttribute); !exists {
					spanAttributes.PutStr(configAction.NewAttribute, configAction.Value)
				} else if _, exists := scopeAttributes.Get(configAction.NewAttribute); !exists {
					spanAttributes.PutStr(configAction.NewAttribute, configAction.Value)
				} else if _, exists := resourceAttributes.Get(configAction.NewAttribute); !exists {
					spanAttributes.PutStr(configAction.NewAttribute, configAction.Value)
				}
			} else if configAction.FromAttribute != "" { // Copy a value from another attribute if specified.
				if fromAttrValue, ok := spanAttributes.Get(configAction.FromAttribute); ok {
					if _, exists := spanAttributes.Get(configAction.NewAttribute); !exists {
						spanAttributes.PutStr(configAction.NewAttribute, fromAttrValue.AsString())
					} else if _, exists := scopeAttributes.Get(configAction.NewAttribute); !exists {
						spanAttributes.PutStr(configAction.NewAttribute, fromAttrValue.AsString())
					} else if _, exists := resourceAttributes.Get(configAction.NewAttribute); !exists {
						spanAttributes.PutStr(configAction.NewAttribute, fromAttrValue.AsString())
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
	if valueConfigActions, exists := rule.Values[scopeName]; exists {
		for _, configAction := range valueConfigActions {
			if configAction.Value != "" {
				// Add static value if not already present
				if _, exists := spanAttributes.Get(configAction.NewAttribute); !exists {
					spanAttributes.PutStr(configAction.NewAttribute, configAction.Value)
				}
			} else if configAction.FromAttribute != "" {
				// Copy value from another attribute if defined
				if fromAttrValue, ok := spanAttributes.Get(configAction.FromAttribute); ok {
					if _, exists := spanAttributes.Get(configAction.NewAttribute); !exists {
						spanAttributes.PutStr(configAction.NewAttribute, fromAttrValue.AsString())
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
