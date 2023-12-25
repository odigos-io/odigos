package googlecloudstorageexporter

import (
	"go.uber.org/zap"
)

type GCSUploadConfig struct {
	GCSBucket    string `mapstructure:"bucket"`
	GCSPrefix    string `mapstructure:"prefix"`
	GCSPartition string `mapstructure:"partition"`
	FilePrefix   string `mapstructure:"file_prefix"`
}

// Config contains the main configuration options for the awskinesis exporter
type Config struct {
	GCSUploader   GCSUploadConfig `mapstructure:"gcs"`
	MarshalerName string          `mapstructure:"marshaler_name"`

	logger *zap.Logger
}
