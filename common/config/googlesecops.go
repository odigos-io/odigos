package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

type GoogleSecOps struct{}

func (g *GoogleSecOps) DestType() common.DestinationType {
	return common.GoogleSecOpsDestinationType
}

func (g *GoogleSecOps) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isLoggingEnabled(dest) {
		return nil, errors.New("Google SecOps requires logs to be enabled")
	}
	fields := dest.GetConfig()
	for _, key := range []string{"GOOGLE_SECOPS_PROJECT", "GOOGLE_SECOPS_CUSTOMER_ID", "GOOGLE_SECOPS_LOCATION", "GOOGLE_SECOPS_LOG_TYPE"} {
		if strings.TrimSpace(fields[key]) == "" {
			return nil, fmt.Errorf("Google SecOps requires %s", key)
		}
	}
	endpoint := fields["GOOGLE_SECOPS_ENDPOINT"]
	if endpoint == "" {
		endpoint = "chronicle.googleapis.com"
	}
	if strings.ContainsAny(endpoint, "/?#") {
		return nil, errors.New("Google SecOps endpoint must be a hostname without a scheme or path")
	}
	exporterName := "chronicle/" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"protocol":              "https",
		"api_version":           "v1",
		"endpoint":              endpoint,
		"override_endpoint":     endpoint != "chronicle.googleapis.com",
		"location":              fields["GOOGLE_SECOPS_LOCATION"],
		"project":               fields["GOOGLE_SECOPS_PROJECT"],
		"customer_id":           fields["GOOGLE_SECOPS_CUSTOMER_ID"],
		"log_type":              fields["GOOGLE_SECOPS_LOG_TYPE"],
		"creds":                 SecretEnvPlaceholder("GOOGLE_SECOPS_CREDENTIALS", dest),
		"namespace":             fields["GOOGLE_SECOPS_NAMESPACE"],
		"compression":           "gzip",
		"timeout":               "90s",
		"collect_agent_metrics": false,
		"retry_on_failure":      GenericMap{"enabled": true},
		"sending_queue":         GenericMap{"enabled": true},
	}
	pipelineName := "logs/googlesecops-" + dest.GetID()
	currentConfig.Service.Pipelines[pipelineName] = Pipeline{Exporters: []string{exporterName}}
	return []string{pipelineName}, nil
}
