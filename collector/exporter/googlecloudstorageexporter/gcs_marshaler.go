package googlecloudstorageexporter

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type GCSMarshaler struct {
	logsMarshaler   plog.Marshaler
	tracesMarshaler ptrace.Marshaler
	logger          *zap.Logger
	format          string
}

func (marshaler *GCSMarshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	return marshaler.tracesMarshaler.MarshalTraces(td)
}

func (marshaler *GCSMarshaler) MarshalLogs(ld plog.Logs) ([]byte, error) {
	return marshaler.logsMarshaler.MarshalLogs(ld)
}

func (marshaler *GCSMarshaler) Format() string {
	return marshaler.format
}
