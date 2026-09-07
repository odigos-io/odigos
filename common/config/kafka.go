package config

import (
	"encoding/json"
	"errors"
	"maps"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	KAFKA_PROTOCOL_VERSION                         = "KAFKA_PROTOCOL_VERSION"
	KAFKA_BROKERS                                  = "KAFKA_BROKERS"
	KAFKA_RESOLVE_CANONICAL_BOOTSTRAP_SERVERS_ONLY = "KAFKA_RESOLVE_CANONICAL_BOOTSTRAP_SERVERS_ONLY"
	KAFKA_CLIENT_ID                                = "KAFKA_CLIENT_ID"
	KAFKA_TOPIC                                    = "KAFKA_TOPIC"
	KAFKA_TOPIC_FROM_ATTRIBUTE                     = "KAFKA_TOPIC_FROM_ATTRIBUTE"
	KAFKA_ENCODING                                 = "KAFKA_ENCODING"
	KAFKA_PARTITION_TRACES_BY_ID                   = "KAFKA_PARTITION_TRACES_BY_ID"
	KAFKA_PARTITION_METRICS_BY_RESOURCE_ATTRIBUTES = "KAFKA_PARTITION_METRICS_BY_RESOURCE_ATTRIBUTES"
	KAFKA_PARTITION_LOGS_BY_RESOURCE_ATTRIBUTES    = "KAFKA_PARTITION_LOGS_BY_RESOURCE_ATTRIBUTES"
	KAFKA_AUTH_METHOD                              = "KAFKA_AUTH_METHOD"
	KAFKA_USERNAME                                 = "KAFKA_USERNAME"
	KAFKA_PASSWORD                                 = "KAFKA_PASSWORD"
	KAFKA_METADATA_FULL                            = "KAFKA_METADATA_FULL"
	KAFKA_METADATA_MAX_RETRY                       = "KAFKA_METADATA_MAX_RETRY"
	KAFKA_METADATA_BACKOFF_RETRY                   = "KAFKA_METADATA_BACKOFF_RETRY"
	KAFKA_TIMEOUT                                  = "KAFKA_TIMEOUT"
	KAFKA_RETRY_ON_FAILURE_ENABLED                 = "KAFKA_RETRY_ON_FAILURE_ENABLED"
	KAFKA_RETRY_ON_FAILURE_INITIAL_INTERVAL        = "KAFKA_RETRY_ON_FAILURE_INITIAL_INTERVAL"
	KAFKA_RETRY_ON_FAILURE_MAX_INTERVAL            = "KAFKA_RETRY_ON_FAILURE_MAX_INTERVAL"
	KAFKA_RETRY_ON_FAILURE_MAX_ELAPSED_TIME        = "KAFKA_RETRY_ON_FAILURE_MAX_ELAPSED_TIME"
	KAFKA_PRODUCER_MAX_MESSAGE_BYTES               = "KAFKA_PRODUCER_MAX_MESSAGE_BYTES"
	KAFKA_PRODUCER_REQUIRED_ACKS                   = "KAFKA_PRODUCER_REQUIRED_ACKS"
	KAFKA_PRODUCER_COMPRESSION                     = "KAFKA_PRODUCER_COMPRESSION"
	KAFKA_PRODUCER_FLUSH_MAX_MESSAGES              = "KAFKA_PRODUCER_FLUSH_MAX_MESSAGES"
)

type Kafka struct{}

func (m *Kafka) DestType() common.DestinationType {
	// DestinationType defined in common/dests.go
	return common.KafkaDestinationType
}

//nolint:funlen,gocyclo // This function is inherently complex due to Kafka config validation, refactoring is non-trivial
func (m *Kafka) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	// To make sure that the exporter and pipeline names are unique, we'll need to define a unique ID
	uniqueUri := "kafka-" + dest.GetID()

	protocolVersion, exists := config[KAFKA_PROTOCOL_VERSION]
	if !exists {
		return nil, errorMissingKey(KAFKA_PROTOCOL_VERSION)
	}
	rawBrokers, exists := config[KAFKA_BROKERS]
	if !exists {
		rawBrokers = "[\"localhost:9092\"]"
	}
	brokers, err := parseKafkaBrokers(rawBrokers)
	if err != nil {
		return nil, err
	}
	resolveCanonicalBootstrapServersOnly, exists := config[KAFKA_RESOLVE_CANONICAL_BOOTSTRAP_SERVERS_ONLY]
	if !exists {
		resolveCanonicalBootstrapServersOnly = "false"
	}
	clientId, exists := config[KAFKA_CLIENT_ID]
	if !exists {
		clientId = "sarama"
	}
	topic, exists := config[KAFKA_TOPIC]
	if !exists {
		topic = "" // defined at bottom of file (otlp_spans, otlp_metrics, otlp_logs)
	}
	topicFromAttribute, exists := config[KAFKA_TOPIC_FROM_ATTRIBUTE]
	if !exists {
		topicFromAttribute = ""
	}
	encoding, exists := config[KAFKA_ENCODING]
	if !exists {
		encoding = "otlp_proto"
	}
	partitionTracesById, exists := config[KAFKA_PARTITION_TRACES_BY_ID]
	if !exists {
		partitionTracesById = "false"
	}
	partitionMetricsByResourceAttributes, exists := config[KAFKA_PARTITION_METRICS_BY_RESOURCE_ATTRIBUTES]
	if !exists {
		partitionMetricsByResourceAttributes = "false"
	}
	partitionLogsByResourceAttributes, exists := config[KAFKA_PARTITION_LOGS_BY_RESOURCE_ATTRIBUTES]
	if !exists {
		partitionLogsByResourceAttributes = "false"
	}
	authMethod, exists := config[KAFKA_AUTH_METHOD]
	if !exists {
		authMethod = "none"
	}
	username, exists := config[KAFKA_USERNAME]
	if !exists {
		username = ""
	}
	metadataFull, exists := config[KAFKA_METADATA_FULL]
	if !exists {
		metadataFull = "false"
	}
	metadataMaxRetry, exists := config[KAFKA_METADATA_MAX_RETRY]
	if !exists {
		metadataMaxRetry = "3"
	}
	metadataBackoffRetry, exists := config[KAFKA_METADATA_BACKOFF_RETRY]
	if !exists {
		metadataBackoffRetry = "250ms"
	}
	timeout, exists := config[KAFKA_TIMEOUT]
	if !exists {
		timeout = "5s"
	}
	retryOnFailureEnabled, exists := config[KAFKA_RETRY_ON_FAILURE_ENABLED]
	if !exists {
		retryOnFailureEnabled = "true"
	}
	retryOnFailureInitialInterval, exists := config[KAFKA_RETRY_ON_FAILURE_INITIAL_INTERVAL]
	if !exists {
		retryOnFailureInitialInterval = "5s"
	}
	retryOnFailureMaxInterval, exists := config[KAFKA_RETRY_ON_FAILURE_MAX_INTERVAL]
	if !exists {
		retryOnFailureMaxInterval = "30s"
	}
	retryOnFailureMaxTimeElapsed, exists := config[KAFKA_RETRY_ON_FAILURE_MAX_ELAPSED_TIME]
	if !exists {
		retryOnFailureMaxTimeElapsed = "120s"
	}
	producerMaxMessageBytes, exists := config[KAFKA_PRODUCER_MAX_MESSAGE_BYTES]
	if !exists {
		producerMaxMessageBytes = "1000000"
	}
	producerRequiredAcks, exists := config[KAFKA_PRODUCER_REQUIRED_ACKS]
	if !exists {
		producerRequiredAcks = "1"
	}
	producerCompression, exists := config[KAFKA_PRODUCER_COMPRESSION]
	if !exists {
		producerCompression = "none"
	}
	producerFlushMaxMessages, exists := config[KAFKA_PRODUCER_FLUSH_MAX_MESSAGES]
	if !exists {
		producerFlushMaxMessages = "0"
	}

	metadataMaxRetryValue, err := parseInt(KAFKA_METADATA_MAX_RETRY, metadataMaxRetry)
	if err != nil {
		return nil, err
	}
	producerMaxMessageBytesValue, err := parseInt(KAFKA_PRODUCER_MAX_MESSAGE_BYTES, producerMaxMessageBytes)
	if err != nil {
		return nil, err
	}
	producerRequiredAcksValue, err := parseInt(KAFKA_PRODUCER_REQUIRED_ACKS, producerRequiredAcks)
	if err != nil {
		return nil, err
	}
	producerFlushMaxMessagesValue, err := parseInt(KAFKA_PRODUCER_FLUSH_MAX_MESSAGES, producerFlushMaxMessages)
	if err != nil {
		return nil, err
	}

	// Modify the exporter here
	exporterName := "kafka/" + uniqueUri
	exporterConfig := GenericMap{
		"protocol_version": protocolVersion,
		"brokers":          brokers,
		"resolve_canonical_bootstrap_servers_only": parseBool(resolveCanonicalBootstrapServersOnly),
		"client_id":              clientId,
		"topic":                  topic,
		"topic_from_attribute":   topicFromAttribute,
		"encoding":               encoding,
		"partition_traces_by_id": parseBool(partitionTracesById),
		"partition_metrics_by_resource_attributes": parseBool(partitionMetricsByResourceAttributes),
		"partition_logs_by_resource_attributes":    parseBool(partitionLogsByResourceAttributes),
		"metadata": GenericMap{
			"full": parseBool(metadataFull),
			"retry": GenericMap{
				"max":     metadataMaxRetryValue,
				"backoff": metadataBackoffRetry,
			},
		},
		"timeout": timeout,
		"retry_on_failure": GenericMap{
			"enabled":          parseBool(retryOnFailureEnabled),
			"initial_interval": retryOnFailureInitialInterval,
			"max_interval":     retryOnFailureMaxInterval,
			"max_elapsed_time": retryOnFailureMaxTimeElapsed,
		},
		"producer": GenericMap{
			"max_message_bytes":  producerMaxMessageBytesValue,
			"required_acks":      producerRequiredAcksValue,
			"compression":        producerCompression,
			"flush_max_messages": producerFlushMaxMessagesValue,
		},
		"auth": GenericMap{
			"tls": GenericMap{
				"insecure": true,
			},
		},
	}

	if authMethod == "plain_text" {
		exporterConfigAuth, ok := exporterConfig["auth"].(GenericMap)
		if !ok {
			return nil, errors.New("invalid type assertion for exporterConfig[\"auth\"]")
		}
		exporterConfigAuth["plain_text"] = GenericMap{
			"username": username,
			"password": "${KAFKA_PASSWORD}",
		}
	}

	// Modify the pipelines here
	var pipelineNames []string

	// when no topic is pinned, each signal exports to a different default topic, so every signal
	// needs its own exporter. Sharing one exporter would leave every pipeline with whichever
	// default topic was written last.
	registerExporter := func(signal string, defaultTopic string) string {
		if topic != "" {
			currentConfig.Exporters[exporterName] = exporterConfig
			return exporterName
		}
		signalExporterName := exporterName + "-" + signal
		signalExporterConfig := maps.Clone(exporterConfig)
		signalExporterConfig["topic"] = defaultTopic
		currentConfig.Exporters[signalExporterName] = signalExporterConfig
		return signalExporterName
	}

	if isTracingEnabled(dest) {
		signalExporterName := registerExporter("traces", "otlp_spans")

		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{signalExporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		signalExporterName := registerExporter("metrics", "otlp_metrics")

		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{signalExporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		signalExporterName := registerExporter("logs", "otlp_logs")

		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{signalExporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}

// KAFKA_BROKERS is a multiInput field, so the UI and the generated docs store it as a
// JSON encoded string array. The kafka exporter expects a real list, and the collector decodes a
// plain string with a comma separated split, so passing the raw value through turns
// `["broker-a:9092","broker-b:9092"]` into the two broker addresses `["broker-a:9092` and
// `"broker-b:9092"]`. Values that are not JSON are still read as a comma separated list, which is
// how a single plain broker address was already being interpreted.
func parseKafkaBrokers(rawBrokers string) ([]string, error) {
	trimmed := strings.TrimSpace(rawBrokers)

	var brokers []string
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &brokers); err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse kafka destination parameter \""+KAFKA_BROKERS+"\" as JSON string in the format: string[]",
			))
		}
	} else {
		brokers = strings.Split(trimmed, ",")
	}

	nonEmptyBrokers := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		if broker = strings.TrimSpace(broker); broker != "" {
			nonEmptyBrokers = append(nonEmptyBrokers, broker)
		}
	}

	// an exporter with no brokers fails to start and would crash loop the whole gateway,
	// so skip this destination instead.
	if len(nonEmptyBrokers) == 0 {
		return nil, errors.New("kafka destination parameter \"" + KAFKA_BROKERS + "\" must list at least one broker")
	}

	return nonEmptyBrokers, nil
}
