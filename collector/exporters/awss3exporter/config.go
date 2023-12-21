package awss3exporter

import (
	"go.uber.org/zap"
)

type AWSS3UploadConfig struct {
	S3Bucket    string `mapstructure:"bucket"`
	S3Region    string `mapstructure:"region"`
	S3Prefix    string `mapstructure:"prefix"`
	S3Partition string `mapstructure:"partition"`
	FilePrefix  string `mapstructure:"file_prefix"`
}

// Config contains the main configuration options for the aws s3 exporter
type Config struct {
	AWSS3UploadConfig AWSS3UploadConfig `mapstructure:"settings"`
	MarshalerName     string            `mapstructure:"marshaler_name"`

	logger *zap.Logger
}
