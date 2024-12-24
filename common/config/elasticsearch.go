package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/odigos-io/odigos/common"
)

const (
	ElasticsearchUrlKey = "ELASTICSEARCH_URL"
	esTracesIndexKey    = "ES_TRACES_INDEX"
	esLogsIndexKey      = "ES_LOGS_INDEX"
	esBasicAuthKey      = "ELASTICSEARCH_BASIC_AUTH_ENABLED" // unused in this file, currently UI only (we do not want to break existing setups by requiring this boolean)
	esUsername          = "ELASTICSEARCH_USERNAME"
	esPassword          = "ELASTICSEARCH_PASSWORD"
	esTlsKey            = "ELASTICSEARCH_TLS_ENABLED" // unused in this file, currently UI only (we do not want to break existing setups by requiring this boolean)
	esCaPem             = "ELASTICSEARCH_CA_PEM"
)

var _ Configer = (*Elasticsearch)(nil)

type Elasticsearch struct{}

func (e *Elasticsearch) DestType() common.DestinationType {
	return common.ElasticsearchDestinationType
}

func (e *Elasticsearch) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	rawURL, exists := dest.GetConfig()[ElasticsearchUrlKey]
	if !exists {
		return errors.New("ElasticSearch url not specified, gateway will not be configured for ElasticSearch")
	}

	parsedURL, err := e.SanitizeURL(rawURL)
	if err != nil {
		return errors.Join(err, errors.New(fmt.Sprintf("failed to sanitize URL. elasticsearch-url: %s", rawURL)))
	}

	traceIndexVal, exists := dest.GetConfig()[esTracesIndexKey]
	if !exists {
		traceIndexVal = "trace_index"
	}

	logIndexVal, exists := dest.GetConfig()[esLogsIndexKey]
	if !exists {
		logIndexVal = "log_index"
	}

	exporterConfig := GenericMap{
		"endpoints":    []string{parsedURL},
		"traces_index": traceIndexVal,
		"logs_index":   logIndexVal,
	}

	caPem := dest.GetConfig()[esCaPem]
	if caPem != "" {
		exporterConfig["tls"] = GenericMap{
			"ca_pem": caPem,
		}
	}

	basicAuthUsername := dest.GetConfig()[esUsername]
	if basicAuthUsername != "" {
		exporterConfig["user"] = basicAuthUsername
		exporterConfig["password"] = fmt.Sprintf("${%s}", esPassword)
	}

	exporterName := "elasticsearch/" + dest.GetID()
	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/elasticsearch-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/elasticsearch-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}

// SanitizeURL will check whether URL is correct by utilizing url.ParseRequestURI
// if the said URL has not defined any port, 9200 will be used in order to keep the backward compatibility with current configuration
func (e *Elasticsearch) SanitizeURL(URL string) (string, error) {
	parsedURL, err := url.ParseRequestURI(URL)
	if err != nil {
		return "", err
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", errors.New("invalid URL")
	}

	if !urlHostContainsPort(parsedURL.Host) {
		parsedURL.Host += ":9200"
	}

	return parsedURL.String(), nil
}
