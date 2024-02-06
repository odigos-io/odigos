package config

import (
	"errors"
	"net/url"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	elasticsearchUrlKey = "ELASTICSEARCH_URL"
	esTracesIndexKey    = "ES_TRACES_INDEX"
	esLogsIndexKey      = "ES_LOGS_INDEX"
)

var _ Configer = (*Elasticsearch)(nil)

type Elasticsearch struct{}

func (e *Elasticsearch) DestType() common.DestinationType {
	return common.ElasticsearchDestinationType
}

func (e *Elasticsearch) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	rawURL, exists := dest.Spec.Data[elasticsearchUrlKey]
	if !exists {
		log.Log.V(0).Info("ElasticSearch url not specified, gateway will not be configured for ElasticSearch")
		return
	}

	parsedURL, err := e.SanitizeURL(rawURL)
	if err != nil {
		log.Log.V(0).Error(err, "failed to sanitize URL", "elasticsearch-url", rawURL)
		return
	}

	if isTracingEnabled(dest) {
		esTraceExporterName := "elasticsearch/trace-" + dest.Name
		traceIndexVal, exists := dest.Spec.Data[esTracesIndexKey]
		if !exists {
			traceIndexVal = "trace_index"
		}

		currentConfig.Exporters[esTraceExporterName] = commonconf.GenericMap{
			"endpoints":    []string{parsedURL},
			"traces_index": traceIndexVal,
		}

		tracesPipelineName := "traces/elasticsearch-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{esTraceExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		esLogExporterName := "elasticsearch/log-" + dest.Name
		logIndexVal, exists := dest.Spec.Data[esLogsIndexKey]
		if !exists {
			logIndexVal = "log_index"
		}

		currentConfig.Exporters[esLogExporterName] = commonconf.GenericMap{
			"endpoints":  []string{parsedURL},
			"logs_index": logIndexVal,
		}

		logsPipelineName := "logs/elasticsearch-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{esLogExporterName},
		}
	}
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
