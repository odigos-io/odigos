package azureblobstorageexporter

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type AzureBlobMarshaler struct {
	logsMarshaler   plog.Marshaler
	tracesMarshaler ptrace.Marshaler
	logger          *zap.Logger
	format          string
}

func (marshaler *AzureBlobMarshaler) MarshalTraces(td ptrace.Traces) ([]byte, error) {
	return marshaler.tracesMarshaler.MarshalTraces(td)
}

func (marshaler *AzureBlobMarshaler) MarshalLogs(ld plog.Logs) ([]byte, error) {
	return marshaler.logsMarshaler.MarshalLogs(ld)
}

func (marshaler *AzureBlobMarshaler) Format() string {
	return marshaler.format
}
