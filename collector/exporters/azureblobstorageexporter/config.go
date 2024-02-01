package azureblobstorageexporter

import (
	"go.uber.org/zap"
)

type AzureBlobStorageUploadConfig struct {
	StorageAccountName string `mapstructure:"account_name"`
	ABSContainer       string `mapstructure:"container"`
	ABSPrefix          string `mapstructure:"prefix"`
	ABSPartition       string `mapstructure:"partition"`
	FilePrefix         string `mapstructure:"file_prefix"`
}

// Config contains the main configuration options for the awskinesis exporter
type Config struct {
	ABSUploader   AzureBlobStorageUploadConfig `mapstructure:"blob"`
	MarshalerName string                       `mapstructure:"marshaler_name"`

	logger *zap.Logger
}
