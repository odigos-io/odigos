package awss3exporter

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type S3Marshaler struct {
	logsMarshaler   plog.Marshaler
	tracesMarshaler ptrace.Marshaler
	logger          *zap.Logger
	format          string
}

func (marshaler *S3Marshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	return marshaler.tracesMarshaler.MarshalTraces(td)
}

func (marshaler *S3Marshaler) MarshalLogs(ld plog.Logs) ([]byte, error) {
	return marshaler.logsMarshaler.MarshalLogs(ld)
}

func (marshaler *S3Marshaler) Format() string {
	return marshaler.format
}
