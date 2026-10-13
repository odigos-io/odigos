package config

import (
	"fmt"
	"maps"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	victoriaTracesEndpoint = "VICTORIA_TRACES_ENDPOINT"
	victoriaTracesAuth     = "VICTORIA_TRACES_AUTH"
	victoriaTracesToken    = "VICTORIA_TRACES_TOKEN"
)

type VictoriaTraces struct{}

type victoriaTracesHTTPDestination struct {
	ExporterConfigurer
	config map[string]string
}

func (d victoriaTracesHTTPDestination) GetConfig() map[string]string { return d.config }

func (v *VictoriaTraces) DestType() common.DestinationType {
	return common.VictoriaTracesDestinationType
}

func (v *VictoriaTraces) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	for _, signal := range dest.GetSignals() {
		if signal != common.TracesObservabilitySignal {
			return nil, fmt.Errorf("VictoriaTraces only supports traces, got %s", signal)
		}
	}
	config := maps.Clone(dest.GetConfig())
	if strings.TrimSpace(config[victoriaTracesEndpoint]) == "" {
		return nil, errorMissingKey(victoriaTracesEndpoint)
	}
	endpoint, err := parseOtlpHttpEndpoint(config[victoriaTracesEndpoint], "", "")
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("VictoriaTraces endpoint must be an HTTP(S) URL with a host and no credentials, query or fragment")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/insert/opentelemetry/v1/traces"
	}
	config[otlpHttpTracesEndpointKey] = parsed.String()
	delete(config, otlpHttpEndpointKey)
	delete(config, otlpHttpMetricsEndpointKey)
	delete(config, otlpHttpLogsEndpointKey)
	delete(config, otlpHttpProfilesEndpointKey)
	delete(config, otlpHttpOAuth2EnabledKey)
	config[otlpHttpTlsKey] = fmt.Sprint(parsed.Scheme == "https")
	if config[otlpHttpCompression] == "" {
		config[otlpHttpCompression] = "gzip"
	}
	if config[otlpHttpCompression] != "gzip" && config[otlpHttpCompression] != "none" {
		return nil, fmt.Errorf("VictoriaTraces compression must be gzip or none")
	}
	auth := config[victoriaTracesAuth]
	switch auth {
	case "", "none", "bearer":
		delete(config, otlpHttpBasicAuthUsernameKey)
	case "basic":
		if strings.TrimSpace(config[otlpHttpBasicAuthUsernameKey]) == "" {
			return nil, errorMissingKey(otlpHttpBasicAuthUsernameKey)
		}
	default:
		return nil, fmt.Errorf("VictoriaTraces authentication must be none, bearer or basic")
	}
	pipelines, err := (&OTLPHttp{}).ModifyConfig(victoriaTracesHTTPDestination{dest, config}, cfg)
	if err != nil {
		return nil, err
	}
	exporterName := "otlp_http/victoriatraces-" + dest.GetID()
	genericExporterName := "otlp_http/generic-" + dest.GetID()
	exporter := cfg.Exporters[genericExporterName].(GenericMap)
	delete(cfg.Exporters, genericExporterName)
	cfg.Exporters[exporterName] = exporter
	if auth == "bearer" {
		extensionName := "bearertokenauth/victoriatraces-" + dest.GetID()
		cfg.Extensions[extensionName] = GenericMap{"token": SecretEnvPlaceholder(victoriaTracesToken, dest)}
		cfg.Service.Extensions = append(cfg.Service.Extensions, extensionName)
		exporter["auth"] = GenericMap{"authenticator": extensionName}
	}
	if len(pipelines) == 0 {
		return nil, nil
	}
	delete(cfg.Service.Pipelines, pipelines[0])
	pipelineName := "traces/victoriatraces-" + dest.GetID()
	cfg.Service.Pipelines[pipelineName] = Pipeline{Exporters: []string{exporterName}}
	return []string{pipelineName}, nil
}
