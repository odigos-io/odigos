package common

import (
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestSpanKindOdigosToOtelClient(t *testing.T) {
	spanKind := ClientSpanKind
	got := SpanKindOdigosToOtel(spanKind)
	want := trace.SpanKindClient
	if got != want {
		t.Errorf("SpanKindOdigosToOtel() = %v, want %v", got, want)
	}
}

func TestSpanKindOdigosToOtelServer(t *testing.T) {
	spanKind := ServerSpanKind
	got := SpanKindOdigosToOtel(spanKind)
	want := trace.SpanKindServer
	if got != want {
		t.Errorf("SpanKindOdigosToOtel() = %v, want %v", got, want)
	}
}

func TestSpanKindOdigosToOtelProducer(t *testing.T) {
	spanKind := ProducerSpanKind
	got := SpanKindOdigosToOtel(spanKind)
	want := trace.SpanKindProducer
	if got != want {
		t.Errorf("SpanKindOdigosToOtel() = %v, want %v", got, want)
	}
}

func TestSpanKindOdigosToOtelConsumer(t *testing.T) {
	spanKind := ConsumerSpanKind
	got := SpanKindOdigosToOtel(spanKind)
	want := trace.SpanKindConsumer
	if got != want {
		t.Errorf("SpanKindOdigosToOtel() = %v, want %v", got, want)
	}
}

func TestSpanKindOdigosToOtelInternal(t *testing.T) {
	spanKind := InternalSpanKind
	got := SpanKindOdigosToOtel(spanKind)
	want := trace.SpanKindInternal
	if got != want {
		t.Errorf("SpanKindOdigosToOtel() = %v, want %v", got, want)
	}
}

func TestSpanKindOdigosToOtelUnspecified(t *testing.T) {
	var spanKind SpanKind
	got := SpanKindOdigosToOtel(spanKind)
	want := trace.SpanKindUnspecified
	if got != want {
		t.Errorf("SpanKindOdigosToOtel() = %v, want %v", got, want)
	}
}
