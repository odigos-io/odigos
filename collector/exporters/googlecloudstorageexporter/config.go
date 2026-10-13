package googlecloudstorageexporter

import (
	"fmt"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type GCSUploadConfig struct {
	GCSBucket    string `mapstructure:"bucket"`
	GCSPrefix    string `mapstructure:"prefix"`
	GCSPartition string `mapstructure:"partition"`
	FilePrefix   string `mapstructure:"file_prefix"`
}

type Config struct {
	GCSUploader   GCSUploadConfig                                          `mapstructure:"gcs"`
	MarshalerName string                                                   `mapstructure:"marshaler_name"`
	TimeoutConfig exporterhelper.TimeoutConfig                             `mapstructure:",squash"`
	QueueConfig   configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`
}

func (c *Config) Validate() error {
	if c.GCSUploader.GCSBucket == "" {
		return fmt.Errorf("gcs.bucket must not be empty")
	}
	if c.GCSUploader.GCSPartition != "minute" && c.GCSUploader.GCSPartition != "hour" {
		return fmt.Errorf("gcs.partition must be minute or hour")
	}
	_, err := NewMarshaler(c.MarshalerName, nil)
	return err
}
