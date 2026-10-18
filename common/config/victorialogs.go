package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

type VictoriaLogs struct{}

func (v *VictoriaLogs) DestType() common.DestinationType {
	return common.VictoriaLogsDestinationType
}

//nolint:gocyclo // destination setup handles endpoint validation, TLS, authentication and ingestion headers
func (v *VictoriaLogs) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	data := dest.GetConfig()
	endpoint, err := parseOtlpHttpEndpoint(data["VICTORIA_LOGS_ENDPOINT"], "", "")
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("VictoriaLogs endpoint must have a host and no credentials or fragment")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/insert/opentelemetry/v1/logs"
	}
	if !strings.HasSuffix(parsed.Path, "/insert/opentelemetry/v1/logs") {
		return nil, fmt.Errorf("VictoriaLogs endpoint must end in /insert/opentelemetry/v1/logs")
	}
	compression := data["VICTORIA_LOGS_COMPRESSION"]
	if compression == "" {
		compression = "gzip"
	}
	if compression != "gzip" && compression != "none" {
		return nil, fmt.Errorf("VictoriaLogs compression must be gzip or none")
	}
	auth := data["VICTORIA_LOGS_AUTH"]
	if auth != "" && auth != "none" && auth != "basic" && auth != "bearer" {
		return nil, fmt.Errorf("VictoriaLogs authentication must be none, basic or bearer")
	}
	if auth == "basic" && data["VICTORIA_LOGS_USERNAME"] == "" {
		return nil, errorMissingKey("VICTORIA_LOGS_USERNAME")
	}
	if !isLoggingEnabled(dest) {
		return nil, nil
	}
	name := "victorialogs-" + dest.GetID()
	exporter := GenericMap{"logs_endpoint": parsed.String(), "compression": compression}
	tls := GenericMap{"insecure": parsed.Scheme == "http"}
	if ca := data["VICTORIA_LOGS_CA_PEM"]; ca != "" {
		tls["ca_pem"] = ca
	}
	tls["insecure_skip_verify"] = parseBool(data["VICTORIA_LOGS_INSECURE_SKIP_VERIFY"])
	exporter["tls"] = tls
	if auth == "basic" || auth == "bearer" {
		extension := "bearertokenauth/" + name
		var authConfig GenericMap
		if auth == "basic" {
			extension = "basicauth/" + name
			authConfig = GenericMap{
				"client_auth": GenericMap{
					"username": data["VICTORIA_LOGS_USERNAME"],
					"password": SecretEnvPlaceholder("VICTORIA_LOGS_PASSWORD", dest),
				},
			}
		} else {
			authConfig = GenericMap{"token": SecretEnvPlaceholder("VICTORIA_LOGS_TOKEN", dest)}
		}
		cfg.Extensions[extension] = authConfig
		cfg.Service.Extensions = append(cfg.Service.Extensions, extension)
		exporter["auth"] = GenericMap{"authenticator": extension}
	}
	for field, header := range map[string]string{
		"ACCOUNT_ID": "AccountID", "PROJECT_ID": "ProjectID", "MSG_FIELD": "VL-Msg-Field", "TIME_FIELD": "VL-Time-Field",
		"STREAM_FIELDS": "VL-Stream-Fields", "IGNORE_FIELDS": "VL-Ignore-Fields", "EXTRA_FIELDS": "VL-Extra-Fields",
		"DECOLORIZE_FIELDS": "VL-Decolorize-Fields", "PRESERVE_JSON_KEYS": "VL-Preserve-JSON-Keys",
	} {
		if value := data["VICTORIA_LOGS_"+field]; value != "" {
			addHeader(exporter, header, value)
		}
	}
	exporterName := "otlp_http/" + name
	cfg.Exporters[exporterName] = exporter
	pipeline := "logs/" + name
	cfg.Service.Pipelines[pipeline] = Pipeline{Exporters: []string{exporterName}}
	return []string{pipeline}, nil
}
