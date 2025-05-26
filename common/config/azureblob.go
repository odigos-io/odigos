package config

import (
	"errors"

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

func (a *AzureBlobStorage) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	accountName, ok := dest.GetConfig()[blobAccountName]
	if !ok {
		return nil, ErrorMissingAzureBlobAccountName
	}

	containerName, ok := dest.GetConfig()[blobContainerName]
	if !ok {
		return nil, ErrorMissingAzureBlobContainerName
	}

	exporterName := "azureblobstorage/" + dest.GetID()
	var pipelineNames []string

	if IsLoggingEnabled(dest) {
		currentConfig.Exporters[exporterName] = GenericMap{
			"blob": GenericMap{
				"account_name": accountName,
				"container":    containerName,
			},
		}

		logsPipelineName := "logs/azureblobstorage-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	if IsTracingEnabled(dest) {
		currentConfig.Exporters[exporterName] = GenericMap{
			"blob": GenericMap{
				"account_name": accountName,
				"container":    containerName,
			},
		}

		tracesPipelineName := "traces/azureblobstorage-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
