package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	AzureDataExplorerClusterURI        = "AZURE_DATA_EXPLORER_CLUSTER_URI"
	AzureDataExplorerDatabase          = "AZURE_DATA_EXPLORER_DATABASE"
	AzureDataExplorerAuthMode          = "AZURE_DATA_EXPLORER_AUTH_MODE"
	AzureDataExplorerApplicationID     = "AZURE_DATA_EXPLORER_APPLICATION_ID"
	AzureDataExplorerApplicationKey    = "AZURE_DATA_EXPLORER_APPLICATION_KEY"
	AzureDataExplorerTenantID          = "AZURE_DATA_EXPLORER_TENANT_ID"
	AzureDataExplorerManagedIdentityID = "AZURE_DATA_EXPLORER_MANAGED_IDENTITY_ID"
	AzureDataExplorerIngestionType     = "AZURE_DATA_EXPLORER_INGESTION_TYPE"
	AzureDataExplorerTracesTable       = "AZURE_DATA_EXPLORER_TRACES_TABLE"
	AzureDataExplorerMetricsTable      = "AZURE_DATA_EXPLORER_METRICS_TABLE"
	AzureDataExplorerLogsTable         = "AZURE_DATA_EXPLORER_LOGS_TABLE"
	AzureDataExplorerTracesMapping     = "AZURE_DATA_EXPLORER_TRACES_MAPPING"
	AzureDataExplorerMetricsMapping    = "AZURE_DATA_EXPLORER_METRICS_MAPPING"
	AzureDataExplorerLogsMapping       = "AZURE_DATA_EXPLORER_LOGS_MAPPING"
)

type AzureDataExplorer struct{}

func (a *AzureDataExplorer) DestType() common.DestinationType {
	return common.AzureDataExplorerDestinationType
}

func (a *AzureDataExplorer) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	cfg := dest.GetConfig()
	endpoint := strings.TrimSpace(cfg[AzureDataExplorerClusterURI])
	if endpoint == "" {
		return nil, errorMissingKey(AzureDataExplorerClusterURI)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%s must be an HTTPS cluster query URI without credentials, path, query or fragment", AzureDataExplorerClusterURI)
	}
	ingestionType := strings.TrimSpace(cfg[AzureDataExplorerIngestionType])
	if ingestionType == "" {
		ingestionType = "queued"
	}
	if ingestionType != "queued" && ingestionType != "managed" {
		return nil, fmt.Errorf("%s must be queued or managed", AzureDataExplorerIngestionType)
	}
	database := strings.TrimSpace(cfg[AzureDataExplorerDatabase])
	if database == "" {
		database = "oteldb"
	}
	exporterConfig := GenericMap{
		"cluster_uri":      strings.TrimSuffix(endpoint, "/"),
		"db_name":          database,
		"ingestion_type":   ingestionType,
		"timeout":          "60s",
		"retry_on_failure": GenericMap{"enabled": true, "initial_interval": "5s", "max_interval": "30s", "max_elapsed_time": "5m"},
		"sending_queue":    GenericMap{"enabled": true, "num_consumers": 2, "queue_size": 1000},
	}
	authMode := strings.TrimSpace(cfg[AzureDataExplorerAuthMode])
	if authMode == "" {
		authMode = "service_principal"
	}
	switch authMode {
	case "service_principal":
		for _, key := range []string{AzureDataExplorerApplicationID, AzureDataExplorerTenantID} {
			if strings.TrimSpace(cfg[key]) == "" {
				return nil, errorMissingKey(key)
			}
		}
		exporterConfig["application_id"] = strings.TrimSpace(cfg[AzureDataExplorerApplicationID])
		exporterConfig["tenant_id"] = strings.TrimSpace(cfg[AzureDataExplorerTenantID])
		exporterConfig["application_key"] = SecretEnvPlaceholder(AzureDataExplorerApplicationKey, dest)
	case "managed_identity":
		identity := strings.TrimSpace(cfg[AzureDataExplorerManagedIdentityID])
		if identity == "" {
			return nil, errorMissingKey(AzureDataExplorerManagedIdentityID)
		}
		exporterConfig["managed_identity_id"] = identity
	case "default":
		exporterConfig["use_azure_auth"] = true
	default:
		return nil, fmt.Errorf("%s must be service_principal, managed_identity or default", AzureDataExplorerAuthMode)
	}
	if isProfilingEnabled(dest) {
		return nil, fmt.Errorf("Azure Data Explorer does not support profiles")
	}
	var pipelineNames []string
	exporterName := "azuredataexplorer/azuredataexplorer-" + dest.GetID()
	for _, signal := range []struct {
		enabled                                  bool
		name, tableKey, mappingKey, defaultTable string
	}{
		{isTracingEnabled(dest), "traces", AzureDataExplorerTracesTable, AzureDataExplorerTracesMapping, "OTELTraces"},
		{isMetricsEnabled(dest), "metrics", AzureDataExplorerMetricsTable, AzureDataExplorerMetricsMapping, "OTELMetrics"},
		{isLoggingEnabled(dest), "logs", AzureDataExplorerLogsTable, AzureDataExplorerLogsMapping, "OTELLogs"},
	} {
		if !signal.enabled {
			continue
		}
		table := strings.TrimSpace(cfg[signal.tableKey])
		if table == "" {
			table = signal.defaultTable
		}
		exporterConfig[signal.name+"_table_name"] = table
		if mapping := strings.TrimSpace(cfg[signal.mappingKey]); mapping != "" {
			exporterConfig[signal.name+"_table_json_mapping"] = mapping
		}
		pipelineNames = append(pipelineNames, signal.name+"/azuredataexplorer-"+dest.GetID())
	}
	currentConfig.Exporters[exporterName] = exporterConfig
	for _, name := range pipelineNames {
		currentConfig.Service.Pipelines[name] = Pipeline{Exporters: []string{exporterName}}
	}
	return pipelineNames, nil
}
