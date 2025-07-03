package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	genericOtlpUrlKey             = "OTLP_GRPC_ENDPOINT"
	genericOtlpTlsKey             = "OTLP_GRPC_TLS_ENABLED"
	genericOtlpCaPemKey           = "OTLP_GRPC_CA_PEM"
	genericOtlpInsecureSkipVerify = "OTLP_GRPC_INSECURE_SKIP_VERIFY"
	otlpGrpcOAuth2EnabledKey      = "OTLP_GRPC_OAUTH2_ENABLED"
	otlpGrpcOAuth2ClientIdKey     = "OTLP_GRPC_OAUTH2_CLIENT_ID"
	otlpGrpcOAuth2ClientSecretKey = "OTLP_GRPC_OAUTH2_CLIENT_SECRET"
	otlpGrpcOAuth2TokenUrlKey     = "OTLP_GRPC_OAUTH2_TOKEN_URL"
	otlpGrpcOAuth2ScopesKey       = "OTLP_GRPC_OAUTH2_SCOPES"
	otlpGrpcOAuth2AudienceKey     = "OTLP_GRPC_OAUTH2_AUDIENCE"
	otlpGrpcCompression           = "OTLP_GRPC_COMPRESSION"
	otlpGrpcHeaders               = "OTLP_GRPC_HEADERS"
)

type GenericOTLP struct{}

func (g *GenericOTLP) DestType() common.DestinationType {
	return common.GenericOTLPDestinationType
}

func (g *GenericOTLP) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	url, exists := config[genericOtlpUrlKey]
	if !exists {
		return nil, errors.New("generic OTLP gRPC endpoint not specified, gateway will not be configured for otlp")
	}

	userTlsEnabled := dest.GetConfig()[genericOtlpTlsKey] == "true"

	// Check for OAuth2 authentication early to determine TLS requirements
	oauth2ExtensionName, oauth2ExtensionConf, err := applyGrpcOAuth2Auth(dest)
	if err != nil {
		return nil, err
	}
	oauth2Enabled := oauth2ExtensionName != ""

	// Determine final TLS setting: gRPC requires TLS when using authentication credentials like OAuth2
	finalTlsEnabled := userTlsEnabled || oauth2Enabled

	grpcEndpoint, err := parseOtlpGrpcUrl(url, finalTlsEnabled)
	if err != nil {
		return nil, errorMissingKey(genericOtlpUrlKey)
	}

	exporterConf := GenericMap{
		"endpoint": grpcEndpoint,
	}

	// Only add TLS config if TLS is needed (user-enabled or OAuth2-required)
	tlsConfig := GenericMap{
		"insecure": !finalTlsEnabled,
	}
	if finalTlsEnabled {
		caPem, caExists := config[genericOtlpCaPemKey]
		if caExists && caPem != "" {
			tlsConfig["ca_pem"] = caPem
		}
		insecureSkipVerify, skipExists := config[genericOtlpInsecureSkipVerify]
		if skipExists && insecureSkipVerify != "" {
			tlsConfig["insecure_skip_verify"] = parseBool(insecureSkipVerify)
		}
	}
	exporterConf["tls"] = tlsConfig

	// add OAuth2 authenticator extension if configured
	if oauth2ExtensionName != "" && oauth2ExtensionConf != nil {
		currentConfig.Extensions[oauth2ExtensionName] = *oauth2ExtensionConf
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, oauth2ExtensionName)
	}

	genericOtlpExporterName := "otlp/generic-" + dest.GetID()

	// Add OAuth2 auth configuration if available
	if oauth2ExtensionName != "" {
		exporterConf["auth"] = GenericMap{
			"authenticator": oauth2ExtensionName,
		}
	}

	if compression, ok := config[otlpGrpcCompression]; ok {
		exporterConf["compression"] = compression
	}

	headers, exists := config[otlpGrpcHeaders]
	if exists && headers != "" {
		var headersList []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		err := json.Unmarshal([]byte(headers), &headersList)
		if err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse otlpGrpc destination OTLP_GRPC_HEADERS parameter as json string in the form {key: string, value: string}[]",
			))
		}
		mappedHeaders := map[string]string{}
		for _, header := range headersList {
			mappedHeaders[header.Key] = header.Value
		}
		exporterConf["headers"] = mappedHeaders
	}
	currentConfig.Exporters[genericOtlpExporterName] = exporterConf

	var pipelineNames []string
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/generic-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{genericOtlpExporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}

func applyGrpcOAuth2Auth(dest ExporterConfigurer) (extensionName string, extensionConf *GenericMap, err error) {
	config := dest.GetConfig()

	oauth2Enabled := config[otlpGrpcOAuth2EnabledKey]
	if oauth2Enabled != "true" {
		return "", nil, nil
	}

	clientId := config[otlpGrpcOAuth2ClientIdKey]
	tokenUrl := config[otlpGrpcOAuth2TokenUrlKey]

	// Note: client secret is stored in the secret and injected as environment variable
	// We don't validate it here since it's not in the regular config data
	if clientId == "" || tokenUrl == "" {
		return "", nil, errors.New("when OAuth2 is enabled, client ID and token URL must be provided")
	}

	extensionName = "oauth2client/otlpgrpc-" + dest.GetID()
	extensionConf = &GenericMap{
		"client_id":     clientId,
		"client_secret": fmt.Sprintf("${%s}", otlpGrpcOAuth2ClientSecretKey),
		"token_url":     tokenUrl,
	}

	// Add optional endpoint parameters
	endpointParams := GenericMap{}

	// Add audience if provided
	if audience := config[otlpGrpcOAuth2AudienceKey]; audience != "" {
		endpointParams["audience"] = audience
	}

	// Add endpoint_params if we have any
	if len(endpointParams) > 0 {
		(*extensionConf)["endpoint_params"] = endpointParams
	}

	// Add scopes if provided
	if scopes := config[otlpGrpcOAuth2ScopesKey]; scopes != "" {
		scopesList := strings.Split(scopes, ",")
		// Trim whitespace from each scope
		for i, scope := range scopesList {
			scopesList[i] = strings.TrimSpace(scope)
		}
		(*extensionConf)["scopes"] = scopesList
	}

	return extensionName, extensionConf, nil
}
