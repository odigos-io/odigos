package config

import (
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	chronosphereCollector = "CHRONOSPHERE_COLLECTOR"
)

type Chronosphere struct{}

func (c *Chronosphere) DestType() common.DestinationType {
	return common.ChronosphereDestinationType
}

func (c *Chronosphere) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[chronosphereCollector]; exists {
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimSuffix(url, ":4317")
		url = strings.TrimSuffix(url, "/remote/write")

		if isTracingEnabled(dest) {
			currentConfig.Exporters["otlp/chronosphere"] = commonconf.GenericMap{
				"endpoint": fmt.Sprintf("%s:4317", url),
				"tls": commonconf.GenericMap{
					"insecure": true,
				},
			}

			currentConfig.Service.Pipelines["traces/chronosphere"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{"otlp/chronosphere"},
			}
		}

		if isMetricsEnabled(dest) {
			currentConfig.Exporters["prometheusremotewrite/chronosphere"] = commonconf.GenericMap{
				"endpoint": fmt.Sprintf("http://%s:3030/remote/write", url),
			}
		}
	}
}
