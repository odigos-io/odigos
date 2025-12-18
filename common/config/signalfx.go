package config

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	signalfxRealm     = "SIGNALFX_REALM"
	signalfxApiUrl    = "SIGNALFX_API_URL"
	signalfxIngestUrl = "SIGNALFX_INGEST_URL"

	SignalfxAccessTokenSecretRefEnabled = "SIGNALFX_ACCESS_TOKEN_SECRET_REF_ENABLED"
	SignalfxAccessTokenSecretName       = "SIGNALFX_ACCESS_TOKEN_SECRET_NAME"
	SignalfxAccessTokenSecretKey        = "SIGNALFX_ACCESS_TOKEN_SECRET_KEY"
	SignalfxAccessTokenDefaultKey       = "SIGNALFX_ACCESS_TOKEN"

	signalfxSendOTLPHistograms      = "SIGNALFX_SEND_OTLP_HISTOGRAMS"
	signalfxSyncHostMetadata        = "SIGNALFX_SYNC_HOST_METADATA"
	signalfxSendingQueueNumConsumer = "SIGNALFX_SENDING_QUEUE_NUM_CONSUMERS"

	signalfxEnableTLS          = "SIGNALFX_ENABLE_TLS"
	signalfxInsecureSkipVerify = "SIGNALFX_INSECURE_SKIP_VERIFY"
	SignalfxCaPemKey           = "SIGNALFX_CA_PEM"
	SignalfxCaConfigMapName    = "SIGNALFX_CA_CONFIGMAP_NAME"
	SignalfxCaConfigMapKey     = "SIGNALFX_CA_CONFIGMAP_KEY"

	signalfxIncludeMetrics            = "SIGNALFX_INCLUDE_METRICS"
	signalfxDisableDefaultTranslation = "SIGNALFX_DISABLE_DEFAULT_TRANSLATION_RULES"

	// Correlation configuration for traces
	signalfxCorrelationEnabled         = "SIGNALFX_CORRELATION_ENABLED"
	signalfxCorrelationEndpoint        = "SIGNALFX_CORRELATION_ENDPOINT"
	signalfxCorrelationTimeout         = "SIGNALFX_CORRELATION_TIMEOUT"
	signalfxCorrelationStaleServiceTTL = "SIGNALFX_CORRELATION_STALE_SERVICE_TIMEOUT"
	signalfxCorrelationMaxRequests     = "SIGNALFX_CORRELATION_MAX_REQUESTS"
	signalfxCorrelationMaxBuffered     = "SIGNALFX_CORRELATION_MAX_BUFFERED"
	signalfxCorrelationMaxRetries      = "SIGNALFX_CORRELATION_MAX_RETRIES"
	signalfxCorrelationLogUpdates      = "SIGNALFX_CORRELATION_LOG_UPDATES"
	signalfxCorrelationRetryDelay      = "SIGNALFX_CORRELATION_RETRY_DELAY"
	signalfxCorrelationCleanupInterval = "SIGNALFX_CORRELATION_CLEANUP_INTERVAL"
	signalfxCorrelationSyncAttributes  = "SIGNALFX_CORRELATION_SYNC_ATTRIBUTES"

	signalfxReceiverOnly                = "SIGNALFX_RECEIVER_ONLY"
	signalfxReceiverEnabled             = "SIGNALFX_RECEIVER_ENABLED"
	signalfxReceiverEndpoint            = "SIGNALFX_RECEIVER_ENDPOINT"
	signalfxReceiverAccessTokenPassthru = "SIGNALFX_RECEIVER_ACCESS_TOKEN_PASSTHROUGH"

	// Receiver-only mode OTLP exporter options
	signalfxReceiverOnlyOTLPEndpoint    = "SIGNALFX_RECEIVER_ONLY_OTLP_ENDPOINT"
	signalfxReceiverOnlyOTLPTLSInsecure = "SIGNALFX_RECEIVER_ONLY_OTLP_TLS_INSECURE"

	signalfxK8sClusterReceiverEnabled = "SIGNALFX_K8S_CLUSTER_RECEIVER_ENABLED"

	// SignalfxCaMountPath is the path where the CA certificate is mounted in the collector pod
	SignalfxCaMountPath            = "/etc/signalfx/certs"
	SignalfxCaSecretVolumeName     = "signalfx-ca-cert"
	SignalfxCaConfigMapVolumeName  = "signalfx-ca-configmap"
	SignalfxCaConfigMapDefaultKey  = "ca.crt"
	SignalfxCaConfigMapMountedFile = "ca-configmap.crt"

	// Default receiver endpoint
	signalfxDefaultReceiverEndpoint = "0.0.0.0:9943"
)

type SignalFx struct{}

func (s *SignalFx) DestType() common.DestinationType {
	return common.SignalFxDestinationType
}

func (s *SignalFx) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	// Check if receiver is enabled and if receiver-only mode is enabled
	// In receiver-only mode, only the SignalFx receiver is configured without the exporter or pipelines
	receiverEnabled := config[signalfxReceiverEnabled] == "true"
	receiverOnly := config[signalfxReceiverOnly] == "true"

	if receiverEnabled && receiverOnly {
		return s.configureReceiverOnlyMode(dest, config, currentConfig), nil
	}

	return s.configureFullDestination(dest, config, currentConfig)
}

// configureReceiverOnlyMode configures just the SignalFx receiver with optional OTLP exporter
func (s *SignalFx) configureReceiverOnlyMode(
	dest ExporterConfigurer,
	config map[string]string,
	currentConfig *Config,
) []string {
	receiverName := s.configureSignalFxReceiver(dest, config, currentConfig)

	// Check if OTLP exporter endpoint is configured
	otlpEndpoint, hasOTLPEndpoint := config[signalfxReceiverOnlyOTLPEndpoint]
	if !hasOTLPEndpoint || otlpEndpoint == "" {
		return nil
	}

	// Configure OTLP exporter
	exporterName := "otlp/signalfx-receiver-" + dest.GetID()
	exporterConfig := GenericMap{
		"endpoint": otlpEndpoint,
	}

	// Add TLS insecure option if enabled
	if tlsInsecure, ok := config[signalfxReceiverOnlyOTLPTLSInsecure]; ok && parseBool(tlsInsecure) {
		exporterConfig["tls"] = GenericMap{
			"insecure": true,
		}
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	// Configure pipeline with receiver and OTLP exporter
	var pipelineNames []string
	pipelineName := "metrics/signalfx-receiver-" + dest.GetID()
	currentConfig.Service.Pipelines[pipelineName] = Pipeline{
		Receivers: []string{receiverName},
		Exporters: []string{exporterName},
	}
	pipelineNames = append(pipelineNames, pipelineName)

	return pipelineNames
}

// configureSignalFxReceiver configures the SignalFx receiver and returns the receiver name
func (s *SignalFx) configureSignalFxReceiver(dest ExporterConfigurer, config map[string]string, currentConfig *Config) string {
	receiverName := "signalfx/" + dest.GetID()
	receiverConfig := GenericMap{}

	// Set endpoint (default to 0.0.0.0:9943)
	endpoint := signalfxDefaultReceiverEndpoint
	if customEndpoint, ok := config[signalfxReceiverEndpoint]; ok && customEndpoint != "" {
		endpoint = customEndpoint
	}
	receiverConfig["endpoint"] = endpoint

	// Set access_token_passthrough if configured
	if tokenPassthrough, ok := config[signalfxReceiverAccessTokenPassthru]; ok && tokenPassthrough != "" {
		receiverConfig["access_token_passthrough"] = parseBool(tokenPassthrough)
	}

	currentConfig.Receivers[receiverName] = receiverConfig
	return receiverName
}

// configureFullDestination configures the full SignalFx destination with exporter and pipelines
func (s *SignalFx) configureFullDestination(dest ExporterConfigurer, config map[string]string, currentConfig *Config) ([]string, error) {
	exporterName := "signalfx/" + dest.GetID()
	exporterConfig := s.buildExporterConfig(config)
	currentConfig.Exporters[exporterName] = exporterConfig

	// Configure receivers
	signalfxReceiverName := ""
	if receiverEnabled, ok := config[signalfxReceiverEnabled]; ok && parseBool(receiverEnabled) {
		signalfxReceiverName = s.configureSignalFxReceiver(dest, config, currentConfig)
	}

	k8sClusterReceiverName := s.configureK8sClusterReceiver(dest, config, currentConfig, exporterName)

	// Configure pipelines
	return s.configurePipelines(dest, currentConfig, exporterName, signalfxReceiverName, k8sClusterReceiverName), nil
}

// buildExporterConfig builds the SignalFx exporter configuration
func (s *SignalFx) buildExporterConfig(config map[string]string) GenericMap {
	realm := config[signalfxRealm]

	apiUrl := fmt.Sprintf("https://api.%s.signalfx.com", realm)
	if customApiUrl, ok := config[signalfxApiUrl]; ok && customApiUrl != "" {
		apiUrl = customApiUrl
	}

	ingestUrl := fmt.Sprintf("https://ingest.%s.signalfx.com", realm)
	if customIngestUrl, ok := config[signalfxIngestUrl]; ok && customIngestUrl != "" {
		ingestUrl = customIngestUrl
	}

	exporterConfig := GenericMap{
		"access_token": "${SIGNALFX_ACCESS_TOKEN}",
		"realm":        realm,
		"api_url":      apiUrl,
		"ingest_url":   ingestUrl,
	}

	s.addOptionalExporterSettings(config, exporterConfig)
	s.addTLSConfig(config, exporterConfig)
	s.addMetricsConfig(config, exporterConfig)
	s.addCorrelationConfig(config, exporterConfig)

	return exporterConfig
}

// addOptionalExporterSettings adds optional settings to the exporter config
func (s *SignalFx) addOptionalExporterSettings(config map[string]string, exporterConfig GenericMap) {
	if sendOTLPHistograms, ok := config[signalfxSendOTLPHistograms]; ok && sendOTLPHistograms != "" {
		exporterConfig["send_otlp_histograms"] = parseBool(sendOTLPHistograms)
	}

	if syncHostMetadata, ok := config[signalfxSyncHostMetadata]; ok && syncHostMetadata != "" {
		exporterConfig["sync_host_metadata"] = parseBool(syncHostMetadata)
	}

	if numConsumers, ok := config[signalfxSendingQueueNumConsumer]; ok && numConsumers != "" {
		exporterConfig["sending_queue"] = GenericMap{
			"num_consumers": parseInt(numConsumers),
		}
	}

	if disableTranslation, ok := config[signalfxDisableDefaultTranslation]; ok && disableTranslation != "" {
		exporterConfig["disable_default_translation_rules"] = parseBool(disableTranslation)
	}
}

// addTLSConfig adds TLS configuration to the exporter config if enabled
func (s *SignalFx) addTLSConfig(config map[string]string, exporterConfig GenericMap) {
	if config[signalfxEnableTLS] != "true" {
		return
	}

	tlsConfig := GenericMap{}

	if insecureSkipVerify, ok := config[signalfxInsecureSkipVerify]; ok && insecureSkipVerify != "" {
		tlsConfig["insecure_skip_verify"] = parseBool(insecureSkipVerify)
	}

	if caPem, ok := config[SignalfxCaPemKey]; ok && caPem != "" {
		tlsConfig["ca_file"] = SignalfxCaMountPath + "/" + SignalfxCaPemKey
	} else if configMapName, ok := config[SignalfxCaConfigMapName]; ok && configMapName != "" {
		tlsConfig["ca_file"] = SignalfxCaMountPath + "/" + SignalfxCaConfigMapMountedFile
	}

	exporterConfig["tls"] = tlsConfig
}

// addMetricsConfig adds metrics-related configuration to the exporter config
func (s *SignalFx) addMetricsConfig(config map[string]string, exporterConfig GenericMap) {
	if includeMetrics, ok := config[signalfxIncludeMetrics]; ok && includeMetrics != "" {
		metricNames := parseMetricNamesList(includeMetrics)
		if len(metricNames) > 0 {
			exporterConfig["include_metrics"] = []GenericMap{
				{"metric_names": metricNames},
			}
		}
	}
}

// addCorrelationConfig adds trace correlation configuration to the exporter config
func (s *SignalFx) addCorrelationConfig(config map[string]string, exporterConfig GenericMap) {
	if config[signalfxCorrelationEnabled] != "true" {
		return
	}

	correlationConfig := GenericMap{}

	if endpoint, ok := config[signalfxCorrelationEndpoint]; ok && endpoint != "" {
		correlationConfig["endpoint"] = endpoint
	}

	if timeout, ok := config[signalfxCorrelationTimeout]; ok && timeout != "" {
		correlationConfig["timeout"] = timeout
	}

	if staleServiceTTL, ok := config[signalfxCorrelationStaleServiceTTL]; ok && staleServiceTTL != "" {
		correlationConfig["stale_service_timeout"] = staleServiceTTL
	}

	if maxRequests, ok := config[signalfxCorrelationMaxRequests]; ok && maxRequests != "" {
		correlationConfig["max_requests"] = parseInt(maxRequests)
	}

	if maxBuffered, ok := config[signalfxCorrelationMaxBuffered]; ok && maxBuffered != "" {
		correlationConfig["max_buffered"] = parseInt(maxBuffered)
	}

	if maxRetries, ok := config[signalfxCorrelationMaxRetries]; ok && maxRetries != "" {
		correlationConfig["max_retries"] = parseInt(maxRetries)
	}

	if logUpdates, ok := config[signalfxCorrelationLogUpdates]; ok && logUpdates != "" {
		correlationConfig["log_updates"] = parseBool(logUpdates)
	}

	if retryDelay, ok := config[signalfxCorrelationRetryDelay]; ok && retryDelay != "" {
		correlationConfig["retry_delay"] = retryDelay
	}

	if cleanupInterval, ok := config[signalfxCorrelationCleanupInterval]; ok && cleanupInterval != "" {
		correlationConfig["cleanup_interval"] = cleanupInterval
	}

	if syncAttributes, ok := config[signalfxCorrelationSyncAttributes]; ok && syncAttributes != "" {
		// Parse as a map of attribute names to source names
		attrs := parseKeyValuePairs(syncAttributes)
		if len(attrs) > 0 {
			correlationConfig["sync_attributes"] = attrs
		}
	}

	exporterConfig["correlation"] = correlationConfig
}

// configureK8sClusterReceiver configures the k8s_cluster receiver if enabled
func (s *SignalFx) configureK8sClusterReceiver(dest ExporterConfigurer, config map[string]string, currentConfig *Config, exporterName string) string {
	k8sClusterEnabled, ok := config[signalfxK8sClusterReceiverEnabled]
	if !ok || !parseBool(k8sClusterEnabled) {
		return ""
	}

	receiverName := "k8s_cluster/signalfx-" + dest.GetID()
	currentConfig.Receivers[receiverName] = GenericMap{
		"auth_type":          "serviceAccount",
		"metadata_exporters": []string{exporterName},
	}
	return receiverName
}

// configurePipelines configures the signal pipelines and returns the pipeline names
func (s *SignalFx) configurePipelines(
	dest ExporterConfigurer,
	currentConfig *Config,
	exporterName, signalfxReceiverName, k8sClusterReceiverName string,
) []string {
	var pipelineNames []string

	if isTracingEnabled(dest) {
		pipelineName := "traces/signalfx-" + dest.GetID()
		pipeline := Pipeline{Exporters: []string{exporterName}}
		if signalfxReceiverName != "" {
			pipeline.Receivers = []string{signalfxReceiverName}
		}
		currentConfig.Service.Pipelines[pipelineName] = pipeline
		pipelineNames = append(pipelineNames, pipelineName)
	}

	if isMetricsEnabled(dest) {
		pipelineName := "metrics/signalfx-" + dest.GetID()
		pipeline := Pipeline{Exporters: []string{exporterName}}
		var receivers []string
		if signalfxReceiverName != "" {
			receivers = append(receivers, signalfxReceiverName)
		}
		if k8sClusterReceiverName != "" {
			receivers = append(receivers, k8sClusterReceiverName)
		}
		if len(receivers) > 0 {
			pipeline.Receivers = receivers
		}
		currentConfig.Service.Pipelines[pipelineName] = pipeline
		pipelineNames = append(pipelineNames, pipelineName)
	}

	if isLoggingEnabled(dest) {
		pipelineName := "logs/signalfx-" + dest.GetID()
		pipeline := Pipeline{Exporters: []string{exporterName}}
		if signalfxReceiverName != "" {
			pipeline.Receivers = []string{signalfxReceiverName}
		}
		currentConfig.Service.Pipelines[pipelineName] = pipeline
		pipelineNames = append(pipelineNames, pipelineName)
	}

	return pipelineNames
}

// parseMetricNamesList parses a comma-separated list of metric names
func parseMetricNamesList(metrics string) []string {
	if metrics == "" {
		return nil
	}
	// Support both comma-separated and newline-separated metric names
	// Replace newlines with commas to normalize the input
	normalized := strings.ReplaceAll(metrics, "\n", ",")
	normalized = strings.ReplaceAll(normalized, "\r", ",")
	parts := strings.Split(normalized, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseKeyValuePairs parses a line-separated list of key=value pairs into a map
func parseKeyValuePairs(input string) map[string]string {
	if input == "" {
		return nil
	}
	result := make(map[string]string)
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				result[key] = value
			}
		}
	}
	return result
}
