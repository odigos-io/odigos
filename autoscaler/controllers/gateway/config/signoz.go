package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"strings"
)

const (
	signozUrlKey = "SIGNOZ_URL"
)

type Signoz struct{}

func (s *Signoz) DestType() common.DestinationType {
	return common.SignozDestinationType
}

func (s *Signoz) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[signozUrlKey]; exists {
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimSuffix(url, ":4317")
		signozExporterName := "otlp/signoz"
		currentConfig.Exporters[signozExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:4317", url),
			"tls": commonconf.GenericMap{
				"insecure": true,
			},
		}
		if isTracingEnabled(dest) {
			currentConfig.Service.Pipelines["traces/signoz"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{signozExporterName},
			}
		}

		if isMetricsEnabled(dest) {
			currentConfig.Service.Pipelines["metrics/signoz"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{signozExporterName},
			}
		}

		if isLoggingEnabled(dest) {
			currentConfig.Service.Pipelines["logs/signoz"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{signozExporterName},
			}
		}
	}
}
