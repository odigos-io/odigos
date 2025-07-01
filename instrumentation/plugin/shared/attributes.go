package shared

import (
	"go.opentelemetry.io/otel/attribute"

	proto "github.com/odigos-io/odigos/instrumentation/plugin/proto/v1"
)

// the following conversion function are from "go.opentelemetry.io/otel/exporters/otlp/otlptrace/internal/tracetransform"

func KeyValues(attrs []attribute.KeyValue) []*proto.KeyValue {
	if len(attrs) == 0 {
		return nil
	}

	out := make([]*proto.KeyValue, 0, len(attrs))
	for _, kv := range attrs {
		out = append(out, &proto.KeyValue{Key: string(kv.Key), Value: Value(kv.Value)})
	}
	return out
}

func Value(v attribute.Value) *proto.AnyValue {
	av := new(proto.AnyValue)
	switch v.Type() {
	case attribute.BOOL:
		av.Value = &proto.AnyValue_BoolValue{
			BoolValue: v.AsBool(),
		}
	case attribute.BOOLSLICE:
		av.Value = &proto.AnyValue_ArrayValue{
			ArrayValue: &proto.ArrayValue{
				Values: boolSliceValues(v.AsBoolSlice()),
			},
		}
	case attribute.INT64:
		av.Value = &proto.AnyValue_IntValue{
			IntValue: v.AsInt64(),
		}
	case attribute.INT64SLICE:
		av.Value = &proto.AnyValue_ArrayValue{
			ArrayValue: &proto.ArrayValue{
				Values: int64SliceValues(v.AsInt64Slice()),
			},
		}
	case attribute.FLOAT64:
		av.Value = &proto.AnyValue_DoubleValue{
			DoubleValue: v.AsFloat64(),
		}
	case attribute.FLOAT64SLICE:
		av.Value = &proto.AnyValue_ArrayValue{
			ArrayValue: &proto.ArrayValue{
				Values: float64SliceValues(v.AsFloat64Slice()),
			},
		}
	case attribute.STRING:
		av.Value = &proto.AnyValue_StringValue{
			StringValue: v.AsString(),
		}
	case attribute.STRINGSLICE:
		av.Value = &proto.AnyValue_ArrayValue{
			ArrayValue: &proto.ArrayValue{
				Values: stringSliceValues(v.AsStringSlice()),
			},
		}
	default:
		av.Value = &proto.AnyValue_StringValue{
			StringValue: "INVALID",
		}
	}
	return av
}


func boolSliceValues(vals []bool) []*proto.AnyValue {
	converted := make([]*proto.AnyValue, len(vals))
	for i, v := range vals {
		converted[i] = &proto.AnyValue{
			Value: &proto.AnyValue_BoolValue{
				BoolValue: v,
			},
		}
	}
	return converted
}

func int64SliceValues(vals []int64) []*proto.AnyValue {
	converted := make([]*proto.AnyValue, len(vals))
	for i, v := range vals {
		converted[i] = &proto.AnyValue{
			Value: &proto.AnyValue_IntValue{
				IntValue: v,
			},
		}
	}
	return converted
}

func float64SliceValues(vals []float64) []*proto.AnyValue {
	converted := make([]*proto.AnyValue, len(vals))
	for i, v := range vals {
		converted[i] = &proto.AnyValue{
			Value: &proto.AnyValue_DoubleValue{
				DoubleValue: v,
			},
		}
	}
	return converted
}

func stringSliceValues(vals []string) []*proto.AnyValue {
	converted := make([]*proto.AnyValue, len(vals))
	for i, v := range vals {
		converted[i] = &proto.AnyValue{
			Value: &proto.AnyValue_StringValue{
				StringValue: v,
			},
		}
	}
	return converted
}

func ToAttributesSlice(kvs []*proto.KeyValue) []attribute.KeyValue {
	if len(kvs) == 0 {
		return nil
	}

	out := make([]attribute.KeyValue, 0, len(kvs))
	for _, kv := range kvs {
		out = append(out, ToAttribute(kv))
	}
	return out
}


func ToAttribute(kv *proto.KeyValue) attribute.KeyValue {
	switch kv.Value.Value.(type) {
	case *proto.AnyValue_BoolValue:
		return attribute.Bool(kv.Key, kv.Value.GetBoolValue())
	case *proto.AnyValue_IntValue:
		return attribute.Int64(kv.Key, kv.Value.GetIntValue())
	case *proto.AnyValue_DoubleValue:
		return attribute.Float64(kv.Key, kv.Value.GetDoubleValue())
	case *proto.AnyValue_StringValue:
		return attribute.String(kv.Key, kv.Value.GetStringValue())
	case *proto.AnyValue_ArrayValue:
		switch kv.Value.GetArrayValue().Values[0].Value.(type) {
		case *proto.AnyValue_BoolValue:
			boolSlice := make([]bool, len(kv.Value.GetArrayValue().Values))
			for i, v := range kv.Value.GetArrayValue().Values {
				boolSlice[i] = v.GetBoolValue()
			}
			return attribute.BoolSlice(kv.Key, boolSlice)
		case *proto.AnyValue_IntValue:
			intSlice := make([]int64, len(kv.Value.GetArrayValue().Values))
			for i, v := range kv.Value.GetArrayValue().Values {
				intSlice[i] = v.GetIntValue()
			}
			return attribute.Int64Slice(kv.Key, intSlice)
		case *proto.AnyValue_DoubleValue:
			floatSlice := make([]float64, len(kv.Value.GetArrayValue().Values))
			for i, v := range kv.Value.GetArrayValue().Values {
				floatSlice[i] = v.GetDoubleValue()
			}
			return attribute.Float64Slice(kv.Key, floatSlice)
		case *proto.AnyValue_StringValue:
			stringSlice := make([]string, len(kv.Value.GetArrayValue().Values))
			for i, v := range kv.Value.GetArrayValue().Values {
				stringSlice[i] = v.GetStringValue()
			}
			return attribute.StringSlice(kv.Key, stringSlice)
		default:
			return attribute.String(kv.Key, "INVALID")
		}

	default:
		return attribute.String(kv.Key, "INVALID")
	}
}