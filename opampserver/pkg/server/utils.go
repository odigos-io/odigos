package server

import (
	"strconv"

	"github.com/odigos-io/odigos/opampserver/protobufs"
)

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
