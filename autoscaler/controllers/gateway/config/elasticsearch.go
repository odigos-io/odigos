package config

import (
	"fmt"

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

type Elasticsearch struct{}

func (e *Elasticsearch) DestType() common.DestinationType {
	return common.ElasticsearchDestinationType
}

func (e *Elasticsearch) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	url, exists := dest.Spec.Data[elasticsearchUrlKey]
	if !exists {
		log.Log.V(0).Info("ElasticSearch url not specified, gateway will not be configured for ElasticSearch")
		return
	}

	if isTracingEnabled(dest) {
		esTraceExporterName := "elasticsearch/trace-" + dest.Name
		traceIndexVal, exists := dest.Spec.Data[esTracesIndexKey]
		if !exists {
			traceIndexVal = "trace_index"
		}

		currentConfig.Exporters[esTraceExporterName] = commonconf.GenericMap{
			"endpoints":    []string{fmt.Sprintf("%s:9200", url)},
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
			"endpoints":  []string{fmt.Sprintf("%s:9200", url)},
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
