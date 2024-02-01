package common

import "go.opentelemetry.io/otel/trace"

// SpanKind is already defined in opentelemetry-go as int.
// this value can go into the CRD in which case it will be string for user convenience.
// +kubebuilder:validation:Enum=client;server;producer;consumer;internal
type SpanKind string

const (
	ClientSpanKind   SpanKind = "client"
	ServerSpanKind   SpanKind = "server"
	ProducerSpanKind SpanKind = "producer"
	ConsumerSpanKind SpanKind = "consumer"
	InternalSpanKind SpanKind = "internal"
)

func SpanKindOdigosToOtel(kind SpanKind) trace.SpanKind {
	switch kind {
	case ClientSpanKind:
		return trace.SpanKindClient
	case ServerSpanKind:
		return trace.SpanKindServer
	case ProducerSpanKind:
		return trace.SpanKindProducer
	case ConsumerSpanKind:
		return trace.SpanKindConsumer
	case InternalSpanKind:
		return trace.SpanKindInternal
	default:
		return trace.SpanKindUnspecified
	}
}

func ConvertSpanKindToString(spanKind trace.SpanKind) SpanKind {
	switch spanKind {
	case trace.SpanKindClient:
		return ClientSpanKind
	case trace.SpanKindServer:
		return ServerSpanKind
	case trace.SpanKindProducer:
		return ProducerSpanKind
	case trace.SpanKindConsumer:
		return ConsumerSpanKind
	case trace.SpanKindInternal:
		return InternalSpanKind
	default:
		return ""
	}
}
