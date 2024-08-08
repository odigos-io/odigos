package googlecloudstorageexporter

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"go.opentelemetry.io/collector/exporter"

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
}

func NewGCSExporter(config *Config,
	params exporter.Settings) (*GCSExporter, error) {

	if config == nil {
		return nil, errors.New("gcs exporter config is nil")
	}

	logger := params.Logger
	expConfig := config
	expConfig.logger = logger

	//validateConfig := expConfig.Validate()
	//
	//if validateConfig != nil {
	//	return nil, validateConfig
	//}

	gcs, err := storage.NewClient(context.Background())
	if err != nil {
		return nil, err
	}

	marshaler, err := NewMarshaler(expConfig.MarshalerName, logger)
	if err != nil {
		return nil, errors.New("unknown marshaler")
	}

	gcsExporter := &GCSExporter{
		config: config,
		dataWriter: &GCSWriter{
			gcsClient: gcs,
		},
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
