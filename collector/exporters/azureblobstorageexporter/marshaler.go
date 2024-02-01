package azureblobstorageexporter

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type Marshaler interface {
	MarshalTraces(td ptrace.Traces) ([]byte, error)
	MarshalLogs(ld plog.Logs) ([]byte, error)
	Format() string
}

var (
	ErrUnknownMarshaler = errors.New("unknown marshaler")
)

func NewMarshaler(name string, logger *zap.Logger) (Marshaler, error) {
	marshaler := &AzureBlobMarshaler{logger: logger}
	switch name {
	case "otlp", "otlp_proto":
		marshaler.logsMarshaler = &plog.ProtoMarshaler{}
		marshaler.tracesMarshaler = &ptrace.ProtoMarshaler{}
		marshaler.format = "proto"
	case "otlp_json":
		marshaler.logsMarshaler = &plog.JSONMarshaler{}
		marshaler.tracesMarshaler = &ptrace.JSONMarshaler{}
		marshaler.format = "json"
	default:
		return nil, ErrUnknownMarshaler
	}
	return marshaler, nil
}
