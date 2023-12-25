package config

import (
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	jaegerUrlKey = "JAEGER_URL"
)

type Jaeger struct{}

func (j *Jaeger) DestType() common.DestinationType {
	return common.JaegerDestinationType
}

func (j *Jaeger) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[jaegerUrlKey]; exists && isTracingEnabled(dest) {
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "https://")

		// Check if url does not contains port
		if !strings.Contains(url, ":") {
			url = fmt.Sprintf("%s:4317", url)
		}

		jaegerExporterName := "otlp/jaeger"
		currentConfig.Exporters[jaegerExporterName] = commonconf.GenericMap{
			"endpoint": url,
			"tls": commonconf.GenericMap{
				"insecure": true,
			},
		}

		currentConfig.Service.Pipelines["traces/jaeger"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{jaegerExporterName},
		}
	}
}
