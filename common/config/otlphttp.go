package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	otlpHttpEndpointKey           = "OTLP_HTTP_ENDPOINT"
	otlpHttpTlsKey                = "OTLP_HTTP_TLS_ENABLED"
	otlpHttpCaPemKey              = "OTLP_HTTP_CA_PEM"
	otlpHttpInsecureSkipVerify    = "OTLP_HTTP_INSECURE_SKIP_VERIFY"
	otlpHttpBasicAuthUsernameKey  = "OTLP_HTTP_BASIC_AUTH_USERNAME"
	otlpHttpBasicAuthPasswordKey  = "OTLP_HTTP_BASIC_AUTH_PASSWORD"
	otlpHttpOAuth2EnabledKey      = "OTLP_HTTP_OAUTH2_ENABLED"
	otlpHttpOAuth2ClientIdKey     = "OTLP_HTTP_OAUTH2_CLIENT_ID"
	otlpHttpOAuth2ClientSecretKey = "OTLP_HTTP_OAUTH2_CLIENT_SECRET"
	otlpHttpOAuth2TokenUrlKey     = "OTLP_HTTP_OAUTH2_TOKEN_URL"
	otlpHttpOAuth2ScopesKey       = "OTLP_HTTP_OAUTH2_SCOPES"
	otlpHttpOAuth2AudienceKey     = "OTLP_HTTP_OAUTH2_AUDIENCE"
	otlpHttpCompression           = "OTLP_HTTP_COMPRESSION"
	otlpHttpHeaders               = "OTLP_HTTP_HEADERS"
)

type OTLPHttp struct{}

func (g *OTLPHttp) DestType() common.DestinationType {
	return common.OtlpHttpDestinationType
}

//nolint:funlen // TODO: make it shorter
func (g *OTLPHttp) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()

	url, exists := config[otlpHttpEndpointKey]
	if !exists {
		return nil, errors.New("OTLP http endpoint not specified, gateway will not be configured for otlp http")
	}

	userTlsEnabled := dest.GetConfig()[otlpHttpTlsKey] == "true"

	parsedUrl, err := parseOtlpHttpEndpoint(url, "", "")
	if err != nil {
		return nil, errors.Join(err, errors.New("otlp http endpoint invalid, gateway will not be configured for otlp http"))
	}

	// Check for OAuth2 or Basic Auth (OAuth2 takes precedence)
	oauth2ExtensionName, oauth2ExtensionConf, err := applyOAuth2Auth(dest)
	if err != nil {
		return nil, err
	}
	basicAuthExtensionName, basicAuthExtensionConf := applyBasicAuth(dest)

	// OAuth2 takes precedence over Basic Auth
	var authExtensionName string
	var authExtensionConf *GenericMap
	hasAuthentication := false
	if oauth2ExtensionName != "" && oauth2ExtensionConf != nil {
		authExtensionName = oauth2ExtensionName
		authExtensionConf = oauth2ExtensionConf
		hasAuthentication = true
	} else if basicAuthExtensionName != "" && basicAuthExtensionConf != nil {
		authExtensionName = basicAuthExtensionName
		authExtensionConf = basicAuthExtensionConf
		hasAuthentication = true
	}

	// add authenticator extension
	if authExtensionName != "" && authExtensionConf != nil {
		currentConfig.Extensions[authExtensionName] = *authExtensionConf
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	}

	otlpHttpExporterName := "otlphttp/generic-" + dest.GetID()
	exporterConf := GenericMap{
		"endpoint": parsedUrl,
	}

	// Only add TLS config if TLS is explicitly enabled or authentication is being used
	tlsConfig := GenericMap{
		"insecure": !userTlsEnabled,
	}
	if userTlsEnabled || hasAuthentication {
		caPem, caExists := config[otlpHttpCaPemKey]
		if caExists && caPem != "" {
			tlsConfig["ca_pem"] = caPem
		}
		insecureSkipVerify, skipExists := config[otlpHttpInsecureSkipVerify]
		if skipExists && insecureSkipVerify != "" {
			tlsConfig["insecure_skip_verify"] = parseBool(insecureSkipVerify)
		}
	}
	exporterConf["tls"] = tlsConfig

	if authExtensionName != "" {
		exporterConf["auth"] = GenericMap{
			"authenticator": authExtensionName,
		}
	}
	if compression, ok := config[otlpHttpCompression]; ok {
		exporterConf["compression"] = compression
	}

	headers, exists := config[otlpHttpHeaders]
	if exists && headers != "" {
		var headersList []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		err := json.Unmarshal([]byte(headers), &headersList)
		if err != nil {
			return nil, errors.Join(err, errors.New(
				"failed to parse otlphttp destination OTLP_HTTP_HEADERS parameter as json string in the form {key: string, value: string}[]",
			))
		}
		mappedHeaders := map[string]string{}
		for _, header := range headersList {
			mappedHeaders[header.Key] = header.Value
		}
		exporterConf["headers"] = mappedHeaders
	}
	currentConfig.Exporters[otlpHttpExporterName] = exporterConf

	var pipelineNames []string
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/otlphttp-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}

func applyOAuth2Auth(dest ExporterConfigurer) (extensionName string, extensionConf *GenericMap, err error) {
	config := dest.GetConfig()

	oauth2Enabled := config[otlpHttpOAuth2EnabledKey]
	if oauth2Enabled != "true" {
		return "", nil, nil
	}

	clientId := config[otlpHttpOAuth2ClientIdKey]
	tokenUrl := config[otlpHttpOAuth2TokenUrlKey]

	// Note: client secret is stored in the secret and injected as environment variable
	// We don't validate it here since it's not in the regular config data
	if clientId == "" || tokenUrl == "" {
		return "", nil, errors.New("when OAuth2 is enabled, client ID and token URL must be provided")
	}

	extensionName = "oauth2client/otlphttp-" + dest.GetID()
	extensionConf = &GenericMap{
		"client_id":     clientId,
		"client_secret": fmt.Sprintf("${%s}", otlpHttpOAuth2ClientSecretKey),
		"token_url":     tokenUrl,
	}

	// Add optional endpoint parameters
	endpointParams := GenericMap{}

	// Add audience if provided
	if audience := config[otlpHttpOAuth2AudienceKey]; audience != "" {
		endpointParams["audience"] = audience
	}

	// Add endpoint_params if we have any
	if len(endpointParams) > 0 {
		(*extensionConf)["endpoint_params"] = endpointParams
	}

	// Add scopes if provided
	if scopes := config[otlpHttpOAuth2ScopesKey]; scopes != "" {
		scopesList := strings.Split(scopes, ",")
		// Trim whitespace from each scope
		for i, scope := range scopesList {
			scopesList[i] = strings.TrimSpace(scope)
		}
		(*extensionConf)["scopes"] = scopesList
	}

	return extensionName, extensionConf, nil
}

func applyBasicAuth(dest ExporterConfigurer) (extensionName string, extensionConf *GenericMap) {
	username := dest.GetConfig()[otlpHttpBasicAuthUsernameKey]
	if username == "" {
		return "", nil
	}

	extensionName = "basicauth/otlphttp-" + dest.GetID()
	extensionConf = &GenericMap{
		"client_auth": GenericMap{
			"username": username,
			"password": fmt.Sprintf("${%s}", otlpHttpBasicAuthPasswordKey),
		},
	}

	return extensionName, extensionConf
}
