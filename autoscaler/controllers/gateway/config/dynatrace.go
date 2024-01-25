package config

import (
	"fmt"
	"net/url"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	dynatraceURLKey      = "DYNATRACE_URL"
	dynatraceAPITOKENKey = "${DYNATRACE_API_TOKEN}"
)

var (
	ErrDynatraceURLNotSpecified      = fmt.Errorf("Dynatrace url  not specified")
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

	exporterName := "otlphttp/dynatrace-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": baseURL + "/api/v2/otlp",
		"headers": commonconf.GenericMap{
			"Authorization": "Api-Token ${DYNATRACE_API_TOKEN}",
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/dynatrace-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/dynatrace-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/dynatrace-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
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

	return fmt.Sprintf("https://%s", u.Host), nil
}
