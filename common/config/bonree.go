package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	BONREE_ENDPOINT       = "BONREE_ENDPOINT"
	BONREE_ACCOUNT_ID     = "BONREE_ACCOUNT_ID"
	BONREE_ENVIRONMENT_ID = "BONREE_ENVIRONMENT_ID"
)

type Bonree struct{}

func (j *Bonree) DestType() common.DestinationType {
	return common.BonreeDestinationType
}

func (j *Bonree) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "bonree-" + dest.GetID()
	var pipelineNames []string

	endpoint, exists := config[BONREE_ENDPOINT]
	if !exists {
		return nil, errorMissingKey(BONREE_ENDPOINT)
	}
	endpoint, err := parseOtlpHttpEndpoint(endpoint, "")
	if err != nil {
		return nil, err
	}

	exporterName := "otlphttp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": endpoint,
		"headers": GenericMap{
			"x-br-acid": "${BONREE_ACCOUNT_ID}",
			"x-br-eid":  "${BONREE_ENVIRONMENT_ID}",
		},
	}

	// Only set connector if it hasn’t already been set:
	if _, ok := cfg.Connectors["servicegraph"]; !ok {
		cfg.Connectors["servicegraph"] = GenericMap{
			"metrics_exporter": exporterName,
			"store": GenericMap{
				"ttl":       "60s",
				"max_items": 100000,
			},
		}
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName, "servicegraph"},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/servicegraph/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Receivers: []string{"servicegraph"},
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
