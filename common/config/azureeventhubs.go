package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	azureEventHubsNamespace        = "AZURE_EVENT_HUBS_NAMESPACE"
	azureEventHubsConnectionString = "AZURE_EVENT_HUBS_CONNECTION_STRING"
	azureEventHubsTracesTopic      = "AZURE_EVENT_HUBS_TRACES_TOPIC"
	azureEventHubsMetricsTopic     = "AZURE_EVENT_HUBS_METRICS_TOPIC"
	azureEventHubsLogsTopic        = "AZURE_EVENT_HUBS_LOGS_TOPIC"
	azureEventHubsEncoding         = "AZURE_EVENT_HUBS_ENCODING"
	azureEventHubsCompression      = "AZURE_EVENT_HUBS_COMPRESSION"
)

var (
	azureEventHubsNamespacePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{4,48}[a-zA-Z0-9]$`)
	azureEventHubsTopicPattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,255}$`)
)

type AzureEventHubs struct{}

type azureEventHubsKafkaDest struct {
	ExporterConfigurer
	config map[string]string
}

func (d azureEventHubsKafkaDest) GetConfig() map[string]string { return d.config }

func (a *AzureEventHubs) DestType() common.DestinationType {
	return common.AzureEventHubsDestinationType
}

func (a *AzureEventHubs) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	namespace := strings.TrimSpace(config[azureEventHubsNamespace])
	if !azureEventHubsNamespacePattern.MatchString(namespace) {
		return nil, fmt.Errorf(
			"%s must be a namespace name of 6-50 letters, digits or hyphens, starting with a letter and ending with a letter or digit",
			azureEventHubsNamespace,
		)
	}
	encoding := config[azureEventHubsEncoding]
	if encoding == "" {
		encoding = "otlp_proto"
	}
	if encoding != "otlp_proto" && encoding != "otlp_json" {
		return nil, fmt.Errorf("%s must be otlp_proto or otlp_json", azureEventHubsEncoding)
	}
	compression := config[azureEventHubsCompression]
	if compression == "" {
		compression = "none"
	}
	if compression != "none" && compression != "gzip" {
		return nil, fmt.Errorf("%s must be none or gzip", azureEventHubsCompression)
	}
	topics := map[string]string{"traces": "otlp_spans", "metrics": "otlp_metrics", "logs": "otlp_logs"}
	for signal, key := range map[string]string{
		"traces": azureEventHubsTracesTopic, "metrics": azureEventHubsMetricsTopic, "logs": azureEventHubsLogsTopic,
	} {
		if topic := strings.TrimSpace(config[key]); topic != "" {
			if !azureEventHubsTopicPattern.MatchString(topic) || topic == "." || topic == ".." {
				return nil, fmt.Errorf("%s must be an event hub name of up to 256 letters, digits, dots, underscores or hyphens", key)
			}
			topics[signal] = topic
		}
	}
	usedTopics := map[string]bool{}
	for _, signal := range dest.GetSignals() {
		topic, supported := topics[strings.ToLower(string(signal))]
		if !supported {
			return nil, fmt.Errorf("Azure Event Hubs does not support signal %s", signal)
		}
		if usedTopics[topic] {
			return nil, fmt.Errorf("Azure Event Hubs requires a separate event hub for each signal")
		}
		usedTopics[topic] = true
	}
	kafkaDest := azureEventHubsKafkaDest{ExporterConfigurer: dest, config: map[string]string{
		KAFKA_PROTOCOL_VERSION: "2.0.0", KAFKA_BROKERS: namespace + ".servicebus.windows.net:9093",
		KAFKA_CLIENT_ID: "odigos", KAFKA_ENCODING: encoding, KAFKA_TIMEOUT: "60s",
		KAFKA_PRODUCER_COMPRESSION: compression, KAFKA_PRODUCER_REQUIRED_ACKS: "-1",
		KAFKA_PRODUCER_FLUSH_MAX_MESSAGES: "1",
	}}
	pipelines, err := (&Kafka{}).modifyConfig(kafkaDest, currentConfig, "azureeventhubs-"+dest.GetID())
	if err != nil {
		return nil, err
	}
	for _, name := range pipelines {
		signal := strings.SplitN(name, "/", 2)[0]
		exporter := currentConfig.Exporters[currentConfig.Service.Pipelines[name].Exporters[0]].(GenericMap)
		// The generic Kafka helper retains legacy fields; Event Hubs uses the bundled exporter's current schema.
		delete(exporter, "topic")
		delete(exporter, "encoding")
		exporter[signal] = GenericMap{"topic": topics[signal], "encoding": encoding}
		exporter["metadata"] = GenericMap{"refresh_interval": "3m"}
		exporter["conn_idle_timeout"] = "3m"
		exporter["tls"] = GenericMap{"insecure": false, "insecure_skip_verify": false, "min_version": "1.2"}
		// Confmap escapes dollar signs with $$ so the SASL username reaches Azure literally.
		exporter["auth"] = GenericMap{"sasl": GenericMap{
			"mechanism": "PLAIN", "version": 1, "username": "$$ConnectionString",
			"password": SecretEnvPlaceholder(azureEventHubsConnectionString, dest),
		}}
		producer := exporter["producer"].(GenericMap)
		// One buffered record keeps multi-partition produce requests below Azure's request limit.
		// v0.159.0 rejects max_broker_write_bytes below 100 MiB, so that limit cannot be used here.
		producer["allow_auto_topic_creation"] = false
	}
	return pipelines, nil
}
