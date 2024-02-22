package config

import (
	"fmt"
	"net/url"

	"sigs.k8s.io/controller-runtime/pkg/log"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	qrynHost      = "QRYN_URL"
	qrynAPIKey    = "QRYN_API_KEY"
	qrynAPISecret = "${QRYN_API_SECRET}"
)

type Qryn struct{}

func (g *Qryn) DestType() common.DestinationType {
	return common.QrynDestinationType
}

func (g *Qryn) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if !g.requiredVarsExists(dest) {
		log.Log.V(0).Info("Qryn config is missing required variables")
		return
	}
	apiKey, apiSecret := g.authData(dest)
	if apiKey == "" || apiSecret == "" {
		log.Log.V(0).Info("Qryn API key or secret not set")
		return
	}

	baseURL, err := parseURL(dest.Spec.Data[qrynHost], apiKey, apiSecret)
	if err != nil {
		log.Log.V(0).Info("Qryn API host is not a valid")
		return
	}

	if isMetricsEnabled(dest) {
		rwExporterName := "prometheusremotewrite/qryn-" + dest.Name
		currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s/api/v1/prom/remote/write", baseURL),
		}
		metricsPipelineName := "metrics/qryn-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{rwExporterName},
		}
	}

	if isTracingEnabled(dest) {
		exporterName := "otlp/qryn-" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s/tempo/spans", baseURL),
		}
		tracesPipelineName := "traces/qryn-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		lokiExporterName := "loki/qryn-" + dest.Name
		currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s/loki/api/v1/push", baseURL),
			"labels": commonconf.GenericMap{
				"attributes": commonconf.GenericMap{
					"k8s.container.name": "k8s_container_name",
					"k8s.pod.name":       "k8s_pod_name",
					"k8s.namespace.name": "k8s_namespace_name",
				},
			},
		}
		logsPipelineName := "logs/qryn-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{lokiExporterName},
		}
	}
}

func (g *Qryn) requiredVarsExists(dest *odigosv1.Destination) bool {
	if _, ok := dest.Spec.Data[qrynHost]; !ok {
		return false
	}
	return true
}

func (g *Qryn) authData(dest *odigosv1.Destination) (string, string) {
	var key string
	if k, ok := dest.Spec.Data[qrynAPIKey]; ok {
		key = k
	}
	return key, qrynAPISecret
}

func parseURL(rawURL, apiKey, apiSecret string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		return parseURL(fmt.Sprintf("https://%s", rawURL), apiKey, apiSecret)
	}

	return fmt.Sprintf("https://%s:%s@%s", apiKey, apiSecret, u.Host), nil
}
