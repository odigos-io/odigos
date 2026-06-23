package mockdestinationexporter

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"go.opentelemetry.io/collector/exporter"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type MockDestinationExporter struct {
	config *Config
	logger *zap.Logger
}

func NewMockDestinationExporter(config *Config,
	params exporter.Settings) (*MockDestinationExporter, error) {

	if config == nil {
		return nil, errors.New("mock destination exporter config is nil")
	}

	logger := params.Logger

	mockDestinationExporter := &MockDestinationExporter{
		config: config,
		logger: logger,
	}
	return mockDestinationExporter, nil
}

func (e *MockDestinationExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e *MockDestinationExporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	var encoded []byte
	switch e.config.Encoding {
	case EncodingProto:
		encoded = e.encode((&ptrace.ProtoMarshaler{}).MarshalTraces(traces))
	case EncodingJSON:
		encoded = e.encode((&ptrace.JSONMarshaler{}).MarshalTraces(traces))
	}
	err := e.mockExport(ctx)
	e.discardEncoded(encoded)
	return err
}

func (e *MockDestinationExporter) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	var encoded []byte
	switch e.config.Encoding {
	case EncodingProto:
		encoded = e.encode((&pmetric.ProtoMarshaler{}).MarshalMetrics(metrics))
	case EncodingJSON:
		encoded = e.encode((&pmetric.JSONMarshaler{}).MarshalMetrics(metrics))
	}
	err := e.mockExport(ctx)
	e.discardEncoded(encoded)
	return err
}

func (e *MockDestinationExporter) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	var encoded []byte
	switch e.config.Encoding {
	case EncodingProto:
		encoded = e.encode((&plog.ProtoMarshaler{}).MarshalLogs(logs))
	case EncodingJSON:
		encoded = e.encode((&plog.JSONMarshaler{}).MarshalLogs(logs))
	}
	err := e.mockExport(ctx)
	e.discardEncoded(encoded)
	return err
}

// encode returns the serialized bytes so they stay referenced for the duration of the
// "send". Real exporters hold the encoded payload in memory until the export completes.
func (e *MockDestinationExporter) encode(encoded []byte, err error) []byte {
	if err != nil {
		e.logger.Warn("mock destination failed to encode telemetry", zap.Error(err))
	}
	return encoded
}

// discardEncoded throws away the serialized bytes once the "send" is done, mirroring a real
// exporter releasing the payload after the export response returns.
func (e *MockDestinationExporter) discardEncoded(encoded []byte) {
	_ = encoded
}

func (e *MockDestinationExporter) mockExport(context.Context) error {
	// not taking care of ctx cancel and shutdown as this is a dummy exporter and not used in production
	<-time.After(e.config.ResponseDuration)
	if rand.Float64() < e.config.RejectFraction {
		return errors.New("export rejected by mock destination")
	}
	return nil
}
