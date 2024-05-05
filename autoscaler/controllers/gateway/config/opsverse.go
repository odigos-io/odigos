package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	opsverseLogsUrl    = "OPSVERSE_LOGS_URL"
	opsverseMetricsUrl = "OPSVERSE_METRICS_URL"
	opsverseTracesUrl  = "OPSVERSE_TRACES_URL"
	opsverseUserName   = "OPSVERSE_USERNAME"
)

type OpsVerse struct{}

func (g *OpsVerse) DestType() common.DestinationType {
	return common.OpsVerseDestinationType
}

func (g *OpsVerse) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	var err error
	if isMetricsEnabled(dest) {
		e := g.isMetricsVarsExists(dest)
		if e != nil {
			err = errors.Join(err, e)
		} else {
			url := fmt.Sprintf("%s/api/v1/write", dest.Spec.Data[opsverseMetricsUrl])
			rwExporterName := "prometheusremotewrite/opsverse-" + dest.Name
			currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
				"endpoint": url,
				"headers": commonconf.GenericMap{
					"Authorization": fmt.Sprintf("Basic %s", "${OPSVERSE_AUTH_TOKEN}"),
				},
			}
	
			metricsPipelineName := "metrics/opsverse-" + dest.Name
			currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
				Exporters: []string{rwExporterName},
			}
		}
	}

	if isTracingEnabled(dest) {
		e := g.isTracingVarsExists(dest)
		if e != nil {
			err = errors.Join(err, e)
		} else {
			url := dest.Spec.Data[opsverseTracesUrl]
			url = strings.TrimPrefix(url, "http://")
			url = strings.TrimPrefix(url, "https://")
			url = fmt.Sprintf("%s:443", url)
			exporterName := "otlp/opsverse-" + dest.Name
			currentConfig.Exporters[exporterName] = commonconf.GenericMap{
				"endpoint": url,
				"headers": commonconf.GenericMap{
					"authorization": "Basic ${OPSVERSE_AUTH_TOKEN}",
				},
			}
	
			currentConfig.Service.Pipelines["traces/opsverse"] = commonconf.Pipeline{
				Exporters: []string{exporterName},
			}
		}
	}

	if isLoggingEnabled(dest) {
		e := g.isLogsVarsExists(dest)
		if e != nil {
			err = errors.Join(err, e)
		} else {
			url := fmt.Sprintf("%s/loki/api/v1/push", dest.Spec.Data[opsverseLogsUrl])

			lokiExporterName := "loki/opsverse-" + dest.Name
			currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
				"endpoint": url,
				"headers": commonconf.GenericMap{
					"Authorization": fmt.Sprintf("Basic %s", "${OPSVERSE_AUTH_TOKEN}"),
				},
				"labels": commonconf.GenericMap{
					"attributes": commonconf.GenericMap{
						"k8s.container.name": "k8s_container_name",
						"k8s.pod.name":       "k8s_pod_name",
						"k8s.namespace.name": "k8s_namespace_name",
					},
				},
			}
	
			logsPipelineName := "logs/opsverse-" + dest.Name
			currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
				Exporters: []string{lokiExporterName},
			}
		}
	}

	return err
}

func (g *OpsVerse) isTracingVarsExists(dest *odigosv1.Destination) error {
	_, exists := dest.Spec.Data[opsverseTracesUrl]
	if !exists {
		return errors.New("OpsVerse OTLP tracing endpoint not specified, gateway will not be configured for tracing")
	}

	_, exists = dest.Spec.Data[opsverseUserName]
	if !exists {
		return errors.New("OpsVerse user not specified, gateway will not be configured for traces")
	}

	return nil
}

func (g *OpsVerse) isLogsVarsExists(dest *odigosv1.Destination) error {
	_, exists := dest.Spec.Data[opsverseLogsUrl]
	if !exists {
		return errors.New("OpsVerse logs endpoint not specified, gateway will not be configured for logs")
	}

	_, exists = dest.Spec.Data[opsverseUserName]
	if !exists {
		return errors.New("OpsVerse user not specified, gateway will not be configured for logs")
	}

	return nil
}

func (g *OpsVerse) isMetricsVarsExists(dest *odigosv1.Destination) error {
	_, exists := dest.Spec.Data[opsverseMetricsUrl]
	if !exists {
		return errors.New("OpsVerse metrics endpoint not specified, gateway will not be configured for metrics")
	}

	_, exists = dest.Spec.Data[opsverseUserName]
	if !exists {
		return errors.New("OpsVerse user not specified, gateway will not be configured for metrics")
	}

	return nil
}
