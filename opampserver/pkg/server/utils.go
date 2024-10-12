package server

import (
	"strconv"

	"github.com/odigos-io/odigos/opampserver/pkg/sdkconfig/configresolvers"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"go.opentelemetry.io/otel/attribute"
)

func opampResourceAttributesToOtel(opampResourceAttributes []configresolvers.ResourceAttribute) []attribute.KeyValue {
	otelAttributes := make([]attribute.KeyValue, 0, len(opampResourceAttributes))
	for _, attr := range opampResourceAttributes {
		// TODO: support any type, not just string
		otelAttributes = append(otelAttributes, attribute.String(attr.Key, attr.Value))
	}
	return otelAttributes
}

func ConvertAnyValueToString(value *protobufs.AnyValue) string {
	switch v := value.Value.(type) {
	case *protobufs.AnyValue_StringValue:
		return v.StringValue
	case *protobufs.AnyValue_IntValue:
		return strconv.FormatInt(v.IntValue, 10)
	case *protobufs.AnyValue_BoolValue:
		if v.BoolValue {
			return "true"
		} else {
			return "false"
		}
	case *protobufs.AnyValue_DoubleValue:
		return strconv.FormatFloat(v.DoubleValue, 'f', -1, 64)
	default:
		return ""
	}
}
