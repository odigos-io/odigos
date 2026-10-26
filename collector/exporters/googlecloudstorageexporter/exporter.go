package googlecloudstorageexporter

import (
	"context"
	"errors"

	"cloud.google.com/go/storage"
	"go.opentelemetry.io/collector/exporter"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type GCSExporter struct {
	config     *Config
	dataWriter DataWriter
	logger     *zap.Logger
	marshaler  Marshaler
	client     *storage.Client
}

func NewGCSExporter(config *Config,
	params exporter.Settings) (*GCSExporter, error) {

	if config == nil {
		return nil, errors.New("gcs exporter config is nil")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}
	logger := params.Logger
	marshaler, err := NewMarshaler(config.MarshalerName, logger)
	if err != nil {
		return nil, err
	}
	gcsExporter := &GCSExporter{
		config:    config,
		logger:    logger,
		marshaler: marshaler,
	}
	return gcsExporter, nil
}

func (e *GCSExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e *GCSExporter) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	buf, err := e.marshaler.MarshalLogs(logs)

	if err != nil {
		return err
	}

	return e.dataWriter.WriteBuffer(ctx, buf, e.config, "logs", e.marshaler.Format())
}

func (e *GCSExporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	buf, err := e.marshaler.MarshalTraces(traces)
	if err != nil {
		return err
	}

	return e.dataWriter.WriteBuffer(ctx, buf, e.config, "traces", e.marshaler.Format())
}

func (e *GCSExporter) Start(ctx context.Context, _ component.Host) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	e.client = client
	e.dataWriter = &GCSWriter{gcsClient: client, logger: e.logger}
	return nil
}

func (e *GCSExporter) Shutdown(context.Context) error {
	if e.client == nil {
		return nil
	}
	return e.client.Close()
}
