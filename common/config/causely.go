package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	causelyUrl = "CAUSELY_URL"
)

type Causely struct{}

func (e *Causely) DestType() common.DestinationType {
	return common.CauselyDestinationType
}

func validateCauselyUrlInput(rawUrl string) (string, error) {
	urlWithScheme := strings.TrimSpace(rawUrl)

	if !strings.Contains(rawUrl, "://") {
		urlWithScheme = "http://" + rawUrl
	}

	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	// Causely does not support paths, so remove it
	parsedUrl.Path = ""

	// --- validate the protocol ---
	// if scheme is https, convert to "http" (Causely does not currently support TLS export)
	if parsedUrl.Scheme == "https" {
		parsedUrl.Scheme = "http"
	}
	// at this point if scheme is not http, it is invalid
	if parsedUrl.Scheme != "http" {
		return "", fmt.Errorf("Causely endpoint scheme must be http, got %s", parsedUrl.Scheme)
	}

	// --- validate host ---
	if parsedUrl.Hostname() == "" {
		return "", fmt.Errorf("Causely endpoint host is required")
	}

	// --- validate port ---
	// allow the user specified port, but fallback to Causely default port 4317 if none provided
	if parsedUrl.Port() == "" {
		parsedUrl.Host = parsedUrl.Hostname() + ":4317"
		if err != nil {
			return "", err
		}
	}

	return parsedUrl.String(), nil
}

func (e *Causely) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	rawUrl, exists := dest.GetConfig()[causelyUrl]
	if !exists {
		return errors.New("Causely url not specified, gateway will not be configured for Causely")
	}

	validatedUrl, err := validateCauselyUrlInput(rawUrl)
	if err != nil {
		return errors.Join(err, errors.New("failed to parse Causely endpoint, gateway will not be configured for Causely"))
	}

	exporterName := "otlp/causely-" + dest.GetID()

	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": validatedUrl,
		"tls": GenericMap{
			"insecure": true,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/causely-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		logsPipelineName := "metrics/causely-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
