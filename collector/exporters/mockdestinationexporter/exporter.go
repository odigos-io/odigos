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
	return e.mockExport(ctx)
}

func (e *MockDestinationExporter) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	return e.mockExport(ctx)
}

func (e *MockDestinationExporter) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	return e.mockExport(ctx)
}

func (e *MockDestinationExporter) mockExport(context.Context) error {
	// not taking care of ctx cancel and shutdown as this is a dummy exporter and not used in production
	<-time.After(e.config.ResponseDuration)
	// check if we should reject the trace by comparing random number with reject fraction
	if rand.Float64() < e.config.RejectFraction {
		return errors.New("export rejected by mock destination")
	}
	return nil
}
