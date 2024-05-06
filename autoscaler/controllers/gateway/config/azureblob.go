package config

import (
	"errors"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
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

func (a *AzureBlobStorage) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	accountName, ok := dest.Spec.Data[blobAccountName]
	if !ok {
		return ErrorMissingAzureBlobAccountName
	}

	containerName, ok := dest.Spec.Data[blobContainerName]
	if !ok {
		return ErrorMissingAzureBlobContainerName
	}

	exporterName := "azureblobstorage/" + dest.Name

	if isLoggingEnabled(dest) {
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"blob": commonconf.GenericMap{
				"account_name": accountName,
				"container":    containerName,
			},
		}

		logsPipelineName := "logs/azureblobstorage-" + dest.Name
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

		tracesPipelineName := "traces/azureblobstorage-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
