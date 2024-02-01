package awss3exporter

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/collector/exporter"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type S3Exporter struct {
	config     *Config
	dataWriter DataWriter
	logger     *zap.Logger
	marshaler  Marshaler
}

func NewS3Exporter(cfg *Config,
	params exporter.CreateSettings) (*S3Exporter, error) {

	if cfg == nil {
		return nil, errors.New("s3 exporter config is nil")
	}

	logger := params.Logger
	expConfig := cfg
	expConfig.logger = logger

	s3Config, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.AWSS3UploadConfig.S3Region))
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(s3Config)

	marshaler, err := NewMarshaler(expConfig.MarshalerName, logger)
	if err != nil {
		return nil, errors.New("unknown marshaler")
	}

	s3Exporter := &S3Exporter{
		config: expConfig,
		dataWriter: &S3Writer{
			s3Client: s3Client,
		},
		logger:    logger,
		marshaler: marshaler,
	}

	return s3Exporter, nil
}

func (e *S3Exporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e *S3Exporter) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	buf, err := e.marshaler.MarshalLogs(logs)

	if err != nil {
		return err
	}

	return e.dataWriter.WriteBuffer(ctx, buf, e.config, "logs", e.marshaler.Format())
}

func (e *S3Exporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	buf, err := e.marshaler.MarshalTraces(traces)
	if err != nil {
		return err
	}

	return e.dataWriter.WriteBuffer(ctx, buf, e.config, "traces", e.marshaler.Format())
}
