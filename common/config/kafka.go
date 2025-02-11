package config

import (
	"errors"

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
	KAFKA_SENDING_QUEUE_ENABLED                    = "KAFKA_SENDING_QUEUE_ENABLED"
	KAFKA_SENDING_QUEUE_NUM_CONSUMERS              = "KAFKA_SENDING_QUEUE_NUM_CONSUMERS"
	KAFKA_SENDING_QUEUE_SIZE                       = "KAFKA_SENDING_QUEUE_SIZE"
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
	brokers, exists := config[KAFKA_BROKERS]
	if !exists {
		brokers = "[\"localhost:9092\"]"
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
	sendingQueueEnabled, exists := config[KAFKA_SENDING_QUEUE_ENABLED]
	if !exists {
		sendingQueueEnabled = "true"
	}
	sendingQueueNumConsumers, exists := config[KAFKA_SENDING_QUEUE_NUM_CONSUMERS]
	if !exists {
		sendingQueueNumConsumers = "10"
	}
	sendingQueueSize, exists := config[KAFKA_SENDING_QUEUE_SIZE]
	if !exists {
		sendingQueueSize = "1000"
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
				"max":     parseInt(metadataMaxRetry),
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
		"sending_queue": GenericMap{
			"enabled":       parseBool(sendingQueueEnabled),
			"num_consumers": parseInt(sendingQueueNumConsumers),
			"queue_size":    parseInt(sendingQueueSize),
		},
		"producer": GenericMap{
			"max_message_bytes":  parseInt(producerMaxMessageBytes),
			"required_acks":      parseInt(producerRequiredAcks),
			"compression":        producerCompression,
			"flush_max_messages": parseInt(producerFlushMaxMessages),
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

	if isTracingEnabled(dest) {
		if topic == "" {
			exporterConfig["topic"] = "otlp_spans"
		}
		currentConfig.Exporters[exporterName] = exporterConfig

		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		if topic == "" {
			exporterConfig["topic"] = "otlp_metrics"
		}
		currentConfig.Exporters[exporterName] = exporterConfig

		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		if topic == "" {
			exporterConfig["topic"] = "otlp_logs"
		}
		currentConfig.Exporters[exporterName] = exporterConfig

		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
