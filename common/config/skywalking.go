package config

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	skyWalkingEndpointKey    = "SKYWALKING_ENDPOINT"
	skyWalkingTLSKey         = "SKYWALKING_TLS_ENABLED"
	skyWalkingCAKey          = "SKYWALKING_CA_PEM"
	skyWalkingAuthKey        = "SKYWALKING_AUTH_ENABLED"
	skyWalkingTokenKey       = "SKYWALKING_TOKEN"
	skyWalkingCompressionKey = "SKYWALKING_COMPRESSION"
)

type SkyWalking struct{}

func (s *SkyWalking) DestType() common.DestinationType {
	return common.SkyWalkingDestinationType
}

func (s *SkyWalking) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	if strings.TrimSpace(config[skyWalkingEndpointKey]) == "" {
		return nil, errorMissingKey(skyWalkingEndpointKey)
	}
	for _, signal := range dest.GetSignals() {
		switch signal {
		case common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal:
		case common.ProfilesObservabilitySignal:
			return nil, errors.New("SkyWalking does not support OTLP profiles")
		default:
			return nil, errors.New("unsupported SkyWalking signal")
		}
	}
	tlsEnabled := config[skyWalkingTLSKey] == "true"
	endpoint, err := parseOtlpGrpcUrlWithDefaultPort(config[skyWalkingEndpointKey], tlsEnabled, "11800")
	if err != nil {
		return nil, err
	}
	compression := config[skyWalkingCompressionKey]
	if compression == "" {
		compression = "none"
	}
	if compression != "none" && compression != "gzip" {
		return nil, errors.New("SkyWalking compression must be none or gzip")
	}
	tlsConfig := GenericMap{"insecure": !tlsEnabled}
	if tlsEnabled && config[skyWalkingCAKey] != "" {
		tlsConfig["ca_pem"] = config[skyWalkingCAKey]
	}
	exporterName := "otlp_grpc/skywalking-" + dest.GetID()
	exporterConfig := GenericMap{
		"endpoint":    endpoint,
		"tls":         tlsConfig,
		"compression": compression,
	}
	if config[skyWalkingAuthKey] == "true" {
		exporterConfig["headers"] = GenericMap{"authentication": SecretEnvPlaceholder(skyWalkingTokenKey, dest)}
	}
	var pipelineNames []string
	for _, signal := range dest.GetSignals() {
		pipelineName := strings.ToLower(string(signal)) + "/skywalking-" + dest.GetID()
		currentConfig.Service.Pipelines[pipelineName] = Pipeline{Exporters: []string{exporterName}}
		pipelineNames = append(pipelineNames, pipelineName)
	}
	currentConfig.Exporters[exporterName] = exporterConfig
	return pipelineNames, nil
}
