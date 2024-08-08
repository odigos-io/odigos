package azureblobstorageexporter

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"go.opentelemetry.io/collector/exporter"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type ABSExporter struct {
	config     *Config
	dataWriter DataWriter
	logger     *zap.Logger
	marshaler  Marshaler
}

func NewAzureBlobExporter(config *Config,
	params exporter.Settings) (*ABSExporter, error) {

	if config == nil {
		return nil, errors.New("azure blob exporter config is nil")
	}

	logger := params.Logger
	expConfig := config
	expConfig.logger = logger

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	ac, err := azblob.NewClient(fmt.Sprintf("https://%s.blob.core.windows.net/", config.ABSUploader.StorageAccountName),
		cred, nil)
	if err != nil {
		return nil, err
	}

	marshaler, err := NewMarshaler(expConfig.MarshalerName, logger)
	if err != nil {
		return nil, errors.New("unknown marshaler")
	}

	azureExporter := &ABSExporter{
		config: config,
		dataWriter: &ABSWriter{
			azureClient: ac,
		},
		logger:    logger,
		marshaler: marshaler,
	}
	return azureExporter, nil
}

func (e *ABSExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e *ABSExporter) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	buf, err := e.marshaler.MarshalLogs(logs)

	if err != nil {
		return err
	}

	return e.dataWriter.WriteBuffer(ctx, buf, e.config, "logs", e.marshaler.Format())
}

func (e *ABSExporter) ConsumeTraces(ctx context.Context, traces ptrace.Traces) error {
	buf, err := e.marshaler.MarshalTraces(traces)
	if err != nil {
		return err
	}

	return e.dataWriter.WriteBuffer(ctx, buf, e.config, "traces", e.marshaler.Format())
}
