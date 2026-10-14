package config

import (
	"fmt"
	"time"

	"github.com/odigos-io/odigos/common"
)

const (
	gmpLocationKey = "GMP_LOCATION"
	gmpClusterKey  = "GMP_CLUSTER"
)

type GoogleManagedPrometheus struct{}

func (g *GoogleManagedPrometheus) DestType() common.DestinationType {
	return common.GoogleManagedPrometheusDestinationType
}

func (g *GoogleManagedPrometheus) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isMetricsEnabled(dest) {
		return nil, fmt.Errorf("Google Cloud Managed Service for Prometheus supports only metrics")
	}
	config := dest.GetConfig()
	if config[gmpLocationKey] == "global" {
		return nil, fmt.Errorf("GMP_LOCATION must be a Google Cloud region or zone; global is not supported")
	}
	exporterConfig := GenericMap{
		"metric": GenericMap{
			"compression": "gzip",
			"resource_filters": []GenericMap{
				{"prefix": "cloud."},
				{"prefix": "k8s."},
				{"prefix": "faas."},
				{"regex": "^(container\\.id|process\\.pid|host\\.name|host\\.id)$"},
			},
		},
		"sending_queue": GenericMap{"enabled": true, "num_consumers": 1},
	}
	if project := config[gcpProjectIdKey]; project != "" {
		exporterConfig["project"] = project
	}
	if timeout := config[gcpTimeoutKey]; timeout != "" {
		duration, err := time.ParseDuration(timeout)
		if err != nil || duration <= 0 {
			return nil, fmt.Errorf("GCP_TIMEOUT must be a positive duration")
		}
		exporterConfig["timeout"] = timeout
	}

	suffix := "googlemanagedprometheus-" + dest.GetID()
	processorNames := []string{}
	resourceAttributes := []GenericMap{}
	for _, field := range []struct{ configKey, attribute string }{
		{gmpLocationKey, "location"},
		{gmpClusterKey, "cluster"},
	} {
		if value := config[field.configKey]; value != "" {
			resourceAttributes = append(resourceAttributes, GenericMap{"key": field.attribute, "value": value, "action": "insert"})
		}
	}
	if len(resourceAttributes) > 0 {
		resourceName := "resource/" + suffix
		currentConfig.Processors[resourceName] = GenericMap{"attributes": resourceAttributes}
		processorNames = append(processorNames, resourceName)
	}

	// GMP drops most resource attributes when identifying a time series. Retain host/process
	// identity for VM metrics and normalize reserved datapoint labels before translation.
	resourceStatements := []string{
		`set(attributes["service.instance.id"], attributes["container.id"]) ` +
			`where attributes["service.instance.id"] == nil and attributes["k8s.pod.name"] == nil and attributes["container.id"] != nil`,
		`set(attributes["service.instance.id"], Concat([attributes["host.id"], String(attributes["process.pid"])], "/")) ` +
			`where attributes["service.instance.id"] == nil and attributes["k8s.pod.name"] == nil ` +
			`and attributes["host.id"] != nil and attributes["process.pid"] != nil`,
		`set(attributes["service.instance.id"], Concat([attributes["host.name"], String(attributes["process.pid"])], "/")) ` +
			`where attributes["service.instance.id"] == nil and attributes["k8s.pod.name"] == nil ` +
			`and attributes["host.name"] != nil and attributes["process.pid"] != nil`,
	}
	datapointStatements := []string{}
	for _, key := range []string{"location", "cluster", "namespace", "job", "instance", "project_id"} {
		datapointStatements = append(datapointStatements,
			fmt.Sprintf(`set(attributes[%q], attributes[%q]) where attributes[%q] != nil`, "exported_"+key, key, key),
			fmt.Sprintf(`delete_key(attributes, %q)`, key))
	}
	transformName := "transform/" + suffix
	currentConfig.Processors[transformName] = GenericMap{
		"error_mode": "propagate",
		"metric_statements": []GenericMap{
			{"context": "resource", "statements": resourceStatements},
			{"context": "datapoint", "statements": datapointStatements},
		},
	}
	processorNames = append(processorNames, transformName)
	exporterName := "googlemanagedprometheus/" + dest.GetID()
	currentConfig.Exporters[exporterName] = exporterConfig
	pipelineName := "metrics/" + suffix
	currentConfig.Service.Pipelines[pipelineName] = Pipeline{Exporters: []string{exporterName}, Processors: processorNames}
	return []string{pipelineName}, nil
}
