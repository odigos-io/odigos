package serviceioconnector

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"

	"github.com/odigos-io/odigos/collector/pkg/completetrace"
)

const spanKindAttribute = "span.kind"

// ExtractSpanAttributes reads configured OpenTelemetry span attribute names from a span node.
// Span kind and instrumentation scope are always included. Each stored key is prefix + attribute name.
// Missing configured attributes and unsupported value types are omitted.
func ExtractSpanAttributes(node *completetrace.TraceTreeNode, prefix string, attributeNames []string) pcommon.Map {
	values := pcommon.NewMap()
	span := node.Span
	scope := node.Scope

	values.PutStr(prefix+spanKindAttribute, span.Kind().String())
	if scopeName := scope.Name(); scopeName != "" {
		values.PutStr(prefix+string(semconv.OTelScopeNameKey), scopeName)
	}
	if scopeVersion := scope.Version(); scopeVersion != "" {
		values.PutStr(prefix+string(semconv.OTelScopeVersionKey), scopeVersion)
	}

	attrs := span.Attributes()
	for _, name := range attributeNames {
		value, ok := attrs.Get(name)
		if !ok || !isSupportedAttributeValue(value) {
			continue
		}
		value.CopyTo(values.PutEmpty(prefix + name))
	}
	return values
}

func isSupportedAttributeValue(value pcommon.Value) bool {
	switch value.Type() {
	case pcommon.ValueTypeStr, pcommon.ValueTypeInt, pcommon.ValueTypeDouble, pcommon.ValueTypeBool:
		return true
	default:
		return false
	}
}

func attributeValueAsString(value pcommon.Value) (string, bool) {
	if !isSupportedAttributeValue(value) {
		return "", false
	}
	return value.AsString(), true
}
