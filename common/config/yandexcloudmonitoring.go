package config

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/odigos-io/odigos/common"
)

const (
	yandexCloudMonitoringWorkspaceIDKey = "YANDEX_CLOUD_MONITORING_WORKSPACE_ID"
	yandexCloudMonitoringAPIKey         = "YANDEX_CLOUD_MONITORING_API_KEY"
)

var yandexCloudMonitoringWorkspaceID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type YandexCloudMonitoring struct{}

func (y *YandexCloudMonitoring) DestType() common.DestinationType {
	return common.YandexCloudMonitoringDestinationType
}

func (y *YandexCloudMonitoring) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	if !isMetricsEnabled(dest) {
		return nil, errors.New("metrics not enabled for Yandex Cloud Monitoring destination")
	}
	workspaceID := dest.GetConfig()[yandexCloudMonitoringWorkspaceIDKey]
	if !yandexCloudMonitoringWorkspaceID.MatchString(workspaceID) {
		return nil, errors.New("Yandex Cloud Monitoring workspace ID must contain only letters, numbers, underscores or hyphens")
	}
	uniqueURI := "yandexcloudmonitoring-" + dest.GetID()
	labels, exists := dest.GetConfig()[prometheusResourceAttributesLabelsKey]
	processors, err := promResourceAttributesProcessors(labels, exists, uniqueURI)
	if err != nil {
		return nil, fmt.Errorf("invalid Yandex Cloud Monitoring resource attribute labels: %w", err)
	}
	authName := "bearertokenauth/" + uniqueURI
	cfg.Extensions[authName] = GenericMap{"token": SecretEnvPlaceholder(yandexCloudMonitoringAPIKey, dest)}
	cfg.Service.Extensions = append(cfg.Service.Extensions, authName)
	exporterName := "prometheusremotewrite/" + uniqueURI
	cfg.Exporters[exporterName] = GenericMap{
		"http": GenericMap{
			"endpoint":    remoteWriteEndpoint("https://monitoring.api.cloud.yandex.net/prometheus/workspaces/" + workspaceID),
			"auth":        GenericMap{"authenticator": authName},
			"compression": "snappy",
		},
		"send_metadata":                 false,
		"protobuf_message":              "prometheus.WriteRequest",
		"translation_strategy":          "UnderscoreEscapingWithSuffixes",
		"target_info":                   GenericMap{"enabled": false},
		"remote_write_queue":            GenericMap{"enabled": true, "num_consumers": 1, "queue_size": 1000},
		"max_batch_request_parallelism": 1,
		// Remote write counts expanded histogram series; the exporter only exposes a byte limit.
		"max_batch_size_bytes": 16000,
		"retry_on_failure":     GenericMap{"enabled": true, "initial_interval": "100ms", "max_interval": "15s", "max_elapsed_time": "300s"},
	}
	filterName := "filter/" + uniqueURI
	cfg.Processors[filterName] = GenericMap{
		"error_mode": "propagate",
		"metrics":    GenericMap{"metric": []string{`type == METRIC_DATA_TYPE_EXPONENTIAL_HISTOGRAM`}},
	}
	processorNames := []string{filterName}
	for name, processor := range processors {
		cfg.Processors[name] = processor
		processorNames = append(processorNames, name)
	}
	pipelineName := "metrics/" + uniqueURI
	cfg.Service.Pipelines[pipelineName] = Pipeline{Processors: processorNames, Exporters: []string{exporterName}}
	return []string{pipelineName}, nil
}
