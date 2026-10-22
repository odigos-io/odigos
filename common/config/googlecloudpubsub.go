package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/odigos-io/odigos/common"
)

const (
	gcpPubSubTopicKey       = "GCP_PUBSUB_TOPIC"
	gcpPubSubCompressionKey = "GCP_PUBSUB_COMPRESSION"
	gcpPubSubEndpointKey    = "GCP_PUBSUB_ENDPOINT"
	gcpPubSubOrderingKey    = "GCP_PUBSUB_ORDERING_KEY"
)

var gcpPubSubTopicPattern = regexp.MustCompile(`^projects/[a-z][a-z0-9-]*/topics/[^/]+$`)

type GoogleCloudPubSub struct{}

func (g *GoogleCloudPubSub) DestType() common.DestinationType {
	return common.GoogleCloudPubSubDestinationType
}

func (g *GoogleCloudPubSub) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isTracingEnabled(dest) && !isMetricsEnabled(dest) && !isLoggingEnabled(dest) {
		return nil, errors.New("GoogleCloudPubSub is not enabled for any supported signals, skipping")
	}
	config := dest.GetConfig()
	topic := strings.TrimSpace(config[gcpPubSubTopicKey])
	if topic == "" {
		return nil, errorMissingKey(gcpPubSubTopicKey)
	}
	if !gcpPubSubTopicPattern.MatchString(topic) {
		return nil, fmt.Errorf("%s must use projects/<project-id>/topics/<topic-id>", gcpPubSubTopicKey)
	}
	exporterConfig := GenericMap{"topic": topic, "project": strings.Split(topic, "/")[1]}
	if project := strings.TrimSpace(config[gcpProjectIdKey]); project != "" {
		exporterConfig["project"] = project
	}
	switch compression := config[gcpPubSubCompressionKey]; compression {
	case "", "none":
	case "gzip":
		exporterConfig["compression"] = compression
	default:
		return nil, fmt.Errorf("%s must be none or gzip", gcpPubSubCompressionKey)
	}
	if timeout := config[gcpTimeoutKey]; timeout != "" {
		duration, err := time.ParseDuration(timeout)
		if err != nil || duration <= 0 {
			return nil, fmt.Errorf("%s must be a positive duration", gcpTimeoutKey)
		}
		exporterConfig["timeout"] = timeout
	}
	if endpoint := strings.TrimSpace(config[gcpPubSubEndpointKey]); endpoint != "" {
		exporterConfig["endpoint"] = endpoint
	}
	if orderingKey := strings.TrimSpace(config[gcpPubSubOrderingKey]); orderingKey != "" {
		exporterConfig["ordering"] = GenericMap{
			"enabled":                 true,
			"from_resource_attribute": orderingKey,
		}
	}
	exporterName := "googlecloudpubsub/" + dest.GetID()
	currentConfig.Exporters[exporterName] = exporterConfig
	var pipelineNames []string
	for _, signal := range []common.ObservabilitySignal{
		common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal,
	} {
		if isSignalExists(dest, signal) {
			name := strings.ToLower(string(signal)) + "/googlecloudpubsub-" + dest.GetID()
			currentConfig.Service.Pipelines[name] = Pipeline{Exporters: []string{exporterName}}
			pipelineNames = append(pipelineNames, name)
		}
	}
	return pipelineNames, nil
}
