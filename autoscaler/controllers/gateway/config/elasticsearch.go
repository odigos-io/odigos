package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
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
	if url, exists := dest.Spec.Data[elasticsearchUrlKey]; exists {
		if isTracingEnabled(dest) {
			esTraceExporterName := "elasticsearch/trace"
			traceIndexVal, exists := dest.Spec.Data[esTracesIndexKey]
			if !exists {
				traceIndexVal = "trace_index"
			}

			currentConfig.Exporters[esTraceExporterName] = commonconf.GenericMap{
				"endpoints":    []string{fmt.Sprintf("%s:9200", url)},
				"traces_index": traceIndexVal,
			}

			currentConfig.Service.Pipelines["traces/elasticsearch"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{esTraceExporterName},
			}
		}

		if isLoggingEnabled(dest) {
			esLogExporterName := "elasticsearch/log"
			logIndexVal, exists := dest.Spec.Data[esLogsIndexKey]
			if !exists {
				logIndexVal = "log_index"
			}

			currentConfig.Exporters[esLogExporterName] = commonconf.GenericMap{
				"endpoints":  []string{fmt.Sprintf("%s:9200", url)},
				"logs_index": logIndexVal,
			}

			currentConfig.Service.Pipelines["logs/elasticsearch"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{esLogExporterName},
			}
		}
	}
}
