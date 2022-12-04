package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

const (
  qrynUrl               = "QRYN_URL"
  qrynUser              = "QRYN_USER"
  qrynToken             = "QRYN_TOKEN"
)

type Qryn struct{}

func (g *Qryn) DestType() common.DestinationType {
	return common.QrynDestinationType
}

func (g *Qryn) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isMetricsEnabled(dest) && g.isQrynVarsExists(dest) {
		url := dest.Spec.Data[qrynUrl]
                if !strings.HasSuffix(url, "/api/prom/remote/write") {
			url = fmt.Sprintf("%s/api/prom/remote/write", url)
		}
		rwExporterName := "prometheusremotewrite/qryn"
		if g.isQrynAuthExists(dest) {
			url = strings.TrimPrefix(dest.Spec.Data[qrynUrl], "https://")
			user := dest.Spec.Data[qrynUser]
                	token := dest.Spec.Data[qrynToken]
  			currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			  "endpoint": fmt.Sprintf("https://%s:%s@%s", user, "${QRYN_TOKEN}", url),
			}
		} else {
			currentConfig.Exporters[rwExporterName] = commonconf.GenericMap{
			  "endpoint": fmt.Sprintf("%s", url),
			}
		}
		currentConfig.Service.Pipelines["metrics/qryn"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{rwExporterName},
		}
	}

	if isTracingEnabled(dest) && g.isQrynVarsExists(dest) {
		url := dest.Spec.Data[qrynUrl]
		if g.isQrynAuthExists(dest) {
			url = strings.TrimPrefix(dest.Spec.Data[qrynUrl], "https://")
			user := dest.Spec.Data[qrynUser]
                	token := dest.Spec.Data[qrynToken]
  			currentConfig.Exporters["otlp/qryn"] = commonconf.GenericMap{
			  "endpoint": fmt.Sprintf("https://%s:%s@%s", user, "${QRYN_TOKEN}", url),
			}
		} else {
			currentConfig.Exporters["otlp/qryn"] = commonconf.GenericMap{
			  "endpoint": fmt.Sprintf("%s", url),
			}
		}
		
		currentConfig.Service.Pipelines["traces/qryn"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/qryn"},
		}
	}

	if isLoggingEnabled(dest) && g.isQrynVarsExists(dest) {
		url := dest.Spec.Data[qrynUrl]
		if !strings.HasSuffix(url, "/loki/api/v1/push") {
			url = fmt.Sprintf("%s/loki/api/v1/push", url)
		}
		lokiExporterName := "loki/qryn"
		if g.isQrynAuthExists(dest) {
			url = strings.TrimPrefix(dest.Spec.Data[qrynUrl], "https://")
			user := dest.Spec.Data[qrynUser]
                	token := dest.Spec.Data[qrynToken]
  			currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
				"endpoint": fmt.Sprintf("https://%s:%s@%s", user, "${QRYN_TOKEN}", url),
				"labels": commonconf.GenericMap{
					"attributes": commonconf.GenericMap{
						"k8s.container.name": "k8s_container_name",
						"k8s.pod.name":       "k8s_pod_name",
						"k8s.namespace.name": "k8s_namespace_name",
					},
				},
			}
		} else {
			currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
				"endpoint": fmt.Sprintf("%s", url),
				"labels": commonconf.GenericMap{
					"attributes": commonconf.GenericMap{
						"k8s.container.name": "k8s_container_name",
						"k8s.pod.name":       "k8s_pod_name",
						"k8s.namespace.name": "k8s_namespace_name",
					},
				},
			}
		}

		currentConfig.Service.Pipelines["logs/qryn"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{lokiExporterName},
		}
	}
}


func (g *Qryn) isQrynVarsExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[qrynUrl]
	if !exists {
		log.Log.V(0).Info("Qryn API URL not specified, gateway will not be configured")
		return false
	}
	
	return true
}

func (g *Qryn) isQrynAuthExists(dest *odigosv1.Destination) bool {
	_, exists := dest.Spec.Data[qrynToken]
	if !exists {
		log.Log.V(0).Info("Qryn API Token not specified, gateway auth will not be configured")
		return false
	}

	_, exists = dest.Spec.Data[qrynUser]
	if !exists {
		log.Log.V(0).Info("Qryn API Auth user not specified, gateway auth will not be configured")
		return false
	}

	return true
}
