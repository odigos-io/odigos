package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	ErrorChronosphereMissingURL = errors.New("missing CHRONOSPHERE_COLLECTOR config")
	ErrorChronosphereNoTls      = errors.New("chronosphere collector url does not support tls")
)

const (
	chronosphereCollector = "CHRONOSPHERE_COLLECTOR"
)

type Chronosphere struct{}

func (c *Chronosphere) DestType() common.DestinationType {
	return common.ChronosphereDestinationType
}

func (c *Chronosphere) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	url, exists := dest.Spec.Data[chronosphereCollector]
	if !exists {
		ctrl.Log.Error(ErrorChronosphereMissingURL, "skipping Chronosphere destination config")
		return
	}

	if strings.HasPrefix(url, "https://") {
		ctrl.Log.Error(ErrorChronosphereNoTls, "skipping Chronosphere destination config")
		return
	}

	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ":4317")
	url = strings.TrimSuffix(url, "/remote/write")

	if isTracingEnabled(dest) {
		chronosphereTraceExporterName := "otlp/chronosphere-" + dest.Name
		currentConfig.Exporters[chronosphereTraceExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:4317", url),
			"tls": commonconf.GenericMap{
				// According to Chronosphere documentation their collector is deployed locally on the cluster
				"insecure": true,
			},
		}

		tracePipelineName := "traces/chronosphere-" + dest.Name
		currentConfig.Service.Pipelines[tracePipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{chronosphereTraceExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		chronosphereMetricsExporterName := "prometheusremotewrite/chronosphere-" + dest.Name
		currentConfig.Exporters[chronosphereMetricsExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("http://%s:3030/remote/write", url),
		}

		metricsPipelineName := "metrics/chronosphere-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{chronosphereMetricsExporterName},
		}
	}
}
