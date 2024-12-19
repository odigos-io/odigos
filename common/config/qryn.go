package config

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	qrynHost                      = "QRYN_URL"
	qrynAPIKey                    = "QRYN_API_KEY"
	qrynAddExporterName           = "QRYN_ADD_EXPORTER_NAME"
	resourceToTelemetryConversion = "QRYN_RESOURCE_TO_TELEMETRY_CONVERSION"
	qrynSecretsOptional           = "__QRYN_SECRETS_OPTIONAL__"
	qrynPasswordFieldName         = "__QRYN_PASSWORD_FIELD_NAME__"
)

type qrynConf struct {
	host                          string
	key                           string
	addExporterName               bool
	resourceToTelemetryConversion bool
	secretsOptional               bool
	passwordFieldName             string
}

type Qryn struct{}

func (g *Qryn) DestType() common.DestinationType {
	return common.QrynDestinationType
}

func (g *Qryn) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	conf := g.getConfigs(dest)
	err := g.checkConfigs(&conf)
	if err != nil {
		return err
	}

	passwordPlaceholder := "${QRYN_API_SECRET}"
	if conf.passwordFieldName != "" {
		passwordPlaceholder = "${" + conf.passwordFieldName + "}"
	}
	baseURL, err := parseURL(conf.host, conf.key, passwordPlaceholder)
	if err != nil {
		return errors.Join(err, errors.New("invalid qryn endpoint. gateway will not be configured with qryn"))
	}

	if isMetricsEnabled(dest) {
		rwExporterName := "prometheusremotewrite/qryn-" + dest.GetID()
		currentConfig.Exporters[rwExporterName] = GenericMap{
			"endpoint": fmt.Sprintf("%s/api/v1/prom/remote/write", baseURL),
			"resource_to_telemetry_conversion": GenericMap{
				"enabled": conf.resourceToTelemetryConversion,
			},
		}
		metricsPipelineName := "metrics/qryn-" + dest.GetID()
		ppl := Pipeline{
			Exporters: []string{rwExporterName},
		}
		g.maybeAddExporterName(
			&conf,
			currentConfig,
			"resource/qryn-metrics-name-"+dest.GetID(),
			"odigos-qryn-metrics",
			&ppl,
		)
		currentConfig.Service.Pipelines[metricsPipelineName] = ppl

	}

	otlpHttpExporterName := ""
	otlpHttpExporter := GenericMap{}
	if isTracingEnabled(dest) {
		otlpHttpExporterName = "otlphttp/qryn-" + dest.GetID()
		otlpHttpExporter["traces_endpoint"] = fmt.Sprintf("%s/v1/traces", baseURL)
		otlpHttpExporter["encoding"] = "proto"
		otlpHttpExporter["compression"] = "none"
		tracesPipelineName := "traces/qryn-" + dest.GetID()
		ppl := Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		g.maybeAddExporterName(
			&conf,
			currentConfig,
			"resource/qryn-traces-name-"+dest.GetID(),
			"odigos-qryn-traces",
			&ppl,
		)
		currentConfig.Service.Pipelines[tracesPipelineName] = ppl

	}

	if isLoggingEnabled(dest) {
		otlpHttpExporterName = "otlphttp/qryn-" + dest.GetID()
		otlpHttpExporter["logs_endpoint"] = fmt.Sprintf("%s/v1/logs", baseURL)
		logsPipelineName := "logs/qryn-" + dest.GetID()
		otlpHttpExporter["encoding"] = "proto"
		otlpHttpExporter["compression"] = "none"
		ppl := Pipeline{
			Exporters: []string{otlpHttpExporterName},
		}
		g.maybeAddExporterName(
			&conf,
			currentConfig,
			"resource/qryn-logs-name-"+dest.GetID(),
			"odigos-qryn-logs",
			&ppl,
		)
		currentConfig.Service.Pipelines[logsPipelineName] = ppl

	}

	if otlpHttpExporterName != "" {
		currentConfig.Exporters[otlpHttpExporterName] = otlpHttpExporter
	}

	return nil
}

func (g *Qryn) getConfigs(dest ExporterConfigurer) qrynConf {
	return qrynConf{
		host:                          dest.GetConfig()[qrynHost],
		key:                           dest.GetConfig()[qrynAPIKey],
		addExporterName:               getBooleanConfig(dest.GetConfig()[qrynAddExporterName], "Yes"),
		resourceToTelemetryConversion: getBooleanConfig(dest.GetConfig()[resourceToTelemetryConversion], "Yes"),
		secretsOptional:               dest.GetConfig()[qrynSecretsOptional] == "1",
		passwordFieldName:             dest.GetConfig()[qrynPasswordFieldName],
	}
}

func (g *Qryn) checkConfigs(conf *qrynConf) error {
	if conf.host == "" {
		return errors.New("missing URL")
	}
	if !conf.secretsOptional && conf.key == "" {
		return errors.New("missing API key")
	}
	return nil
}

func parseURL(rawURL, apiKey, apiSecret string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	apiSecretPlaceholder := fmt.Sprintf("____%d_SECRET_PLACEHOLDER_%[1]d____", rand.Uint64())
	if apiKey != "" {
		u.User = url.UserPassword(apiKey, apiSecretPlaceholder)
	}
	res := u.String()
	if apiKey != "" {
		res = strings.ReplaceAll(res, ":"+apiSecretPlaceholder+"@", ":"+apiSecret+"@")
	}
	return res, nil
}

func (g *Qryn) maybeAddExporterName(conf *qrynConf, currentConfig *Config, processorName string, name string,
	pipeline *Pipeline) {
	if !conf.addExporterName {
		return
	}
	currentConfig.Processors[processorName] = GenericMap{
		"attributes": []GenericMap{
			{
				"action": "upsert",
				"key":    "qryn_exporter",
				"value":  name,
			},
		},
	}
	pipeline.Processors = append(pipeline.Processors, processorName)
}
