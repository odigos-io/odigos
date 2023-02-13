package config

import (
	"errors"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	ctrl "sigs.k8s.io/controller-runtime"
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

func (a *AzureBlobStorage) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	accountName, ok := dest.Spec.Data[blobAccountName]
	if !ok {
		ctrl.Log.Error(ErrorMissingAzureBlobAccountName, "skipping Azure Blob Storage config")
		return
	}

	containerName, ok := dest.Spec.Data[blobContainerName]
	if !ok {
		ctrl.Log.Error(ErrorMissingAzureBlobContainerName, "skipping Azure Blob Storage config")
		return
	}

	if isLoggingEnabled(dest) {
		currentConfig.Exporters["azureblobstorage"] = commonconf.GenericMap{
			"blob": commonconf.GenericMap{
				"account_name": accountName,
				"container":    containerName,
			},
		}

		currentConfig.Service.Pipelines["logs/azureblobstorage"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"azureblobstorage"},
		}
	}

	if isTracingEnabled(dest) {
		currentConfig.Exporters["azureblobstorage"] = commonconf.GenericMap{
			"blob": commonconf.GenericMap{
				"account_name": accountName,
				"container":    containerName,
			},
		}

		currentConfig.Service.Pipelines["traces/azureblobstorage"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"azureblobstorage"},
		}
	}
}
