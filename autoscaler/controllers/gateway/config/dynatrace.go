package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
    "sigs.k8s.io/controller-runtime/pkg/log"
	"fmt"
	"net/url"
)

const (
	dynatraceURLKey = "DYNATRACE_URL"
	dynatraceAPITOKENKey = "${DYNATRACE_API_TOKEN}"
)

var (
	ErrDynatraceURLNotSpecified = fmt.Errorf("Dynatrace url  not specified")
	ErrDynatraceAPITOKENNotSpecified = fmt.Errorf("Api token not specified")
)

type Dynatrace struct{}

func (n *Dynatrace) DestType() common.DestinationType {
	return common.DynatraceDestinationType
}

func (n *Dynatrace) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

   if !n.requiredVarsExists(dest) {
   		log.Log.V(0).Info("Dynatrace config is missing required variables")
   		return
   	}

   	baseURL, err := parsetheDTurl(dest.Spec.Data[dynatraceURLKey])
   	if err != nil {
   		log.Log.V(0).Info("Dynatrace url is not a valid")
   		return
   	}

	currentConfig.Exporters["otlphttp/dynatrace"] = commonconf.GenericMap{
		"endpoint": baseURL +"/api/v2/otlp",
		"headers": commonconf.GenericMap{
			"Authorization": "Api-Token ${DYNATRACE_API_TOKEN}",
		},
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/dynatrace"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/dynatrace"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/dynatrace"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/dynatrace"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/dynatrace"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/dynatrace"},
		}
	}
}
func (g *Dynatrace) requiredVarsExists(dest *odigosv1.Destination) bool {
	if _, ok := dest.Spec.Data[dynatraceURLKey]; !ok {
		return false
	}
	return true
}



func parsetheDTurl(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		return parsetheDTurl(fmt.Sprintf("https://%s", rawURL))
	}

	return fmt.Sprintf("https://%s",  u.Host), nil
}