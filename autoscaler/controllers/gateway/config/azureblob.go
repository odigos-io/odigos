package config

import (
	"errors"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	blobAccountName   = "AZURE_BLOB_ACCOUNT_NAME"
	blobContainerName = "AZURE_BLOB_CONTAINER_NAME"
)

var (
	ErrorMissingAzureBlobAccountName   = errors.New("missing Azure Blob Account Name")
	ErrorMissingAzureBlobContainerName = errors.New("missing Azure Blob Container Name")
)

type AzureBlobStorage struct{}

func (a *AzureBlobStorage) DestType() common.DestinationType {
	return common.AzureBlobDestinationType
}

func (a *AzureBlobStorage) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {
	accountName, ok := dest.GetConfig()[blobAccountName]
	if !ok {
		return ErrorMissingAzureBlobAccountName
	}

	containerName, ok := dest.GetConfig()[blobContainerName]
	if !ok {
		return ErrorMissingAzureBlobContainerName
	}

	exporterName := "azureblobstorage/" + dest.GetName()

	if isLoggingEnabled(dest) {
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"blob": commonconf.GenericMap{
				"account_name": accountName,
				"container":    containerName,
			},
		}

		logsPipelineName := "logs/azureblobstorage-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isTracingEnabled(dest) {
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"blob": commonconf.GenericMap{
				"account_name": accountName,
				"container":    containerName,
			},
		}

		tracesPipelineName := "traces/azureblobstorage-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
