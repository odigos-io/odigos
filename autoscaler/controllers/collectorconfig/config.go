package collectorconfig

import (
	"fmt"
	"github.com/ghodss/yaml"
	v1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"strings"
)

type genericMap map[string]interface{}

type Config struct {
	Receivers  genericMap `json:"receivers"`
	Exporters  genericMap `json:"exporters"`
	Processors genericMap `json:"processors"`
	Extensions genericMap `json:"extensions"`
	Service    genericMap `json:"service"`
}

func getExportersForDest(dst *v1.Destination) genericMap {
	if dst.Spec.Type == v1.GrafanaDestinationType {
		//authString := fmt.Sprintf("%s:%s", dst.Spec.Data.Grafana.User, dst.Spec.Data.Grafana.ApiKey)
		//encodedAuthString := b64.StdEncoding.EncodeToString([]byte(authString))
		url := strings.TrimSuffix(dst.Spec.Data.Grafana.Url, "/tempo")
		return genericMap{
			"otlp": genericMap{
				"endpoint": fmt.Sprintf("%s:%d", url, 443),
				"headers": genericMap{
					"authorization": "Basic ${AUTH_TOKEN}",
				},
			},
		}
	} else if dst.Spec.Type == v1.HoneycombDestinationType {
		return genericMap{
			"otlp": genericMap{
				"endpoint": "api.honeycomb.io:443",
				"headers": genericMap{
					"x-honeycomb-team": "${API_KEY}",
				},
			},
		}
	} else if dst.Spec.Type == v1.DatadogDestinationType {
		return genericMap{
			"datadog": genericMap{
				"api": genericMap{
					"key":  "${API_KEY}",
					"site": dst.Spec.Data.Datadog.Site,
				},
			},
		}
	} else if dst.Spec.Type == v1.NewRelicDestinationType {
		return genericMap{
			"otlp": genericMap{
				"endpoint": "https://otlp.nr-data.net:4317",
				"headers": genericMap{
					"api-key": "${API_KEY}",
				},
			},
		}
	}

	return genericMap{}
}

func getExporters(dest *v1.DestinationList) genericMap {
	for _, dst := range dest.Items {
		return getExportersForDest(&dst)
	}

	return genericMap{}
}

func GetConfigForCollector(dests *v1.DestinationList) (string, error) {
	empty := struct{}{}
	exporters := getExporters(dests)
	c := &Config{
		Receivers: genericMap{
			"zipkin": empty,
			"otlp": genericMap{
				"protocols": genericMap{
					"grpc": empty,
					"http": empty,
				},
			},
		},
		Exporters: exporters,
		Processors: genericMap{
			"batch": empty,
		},
		Extensions: genericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Service: getService(dests),
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getService(dests *v1.DestinationList) genericMap {
	return genericMap{
		"extensions": []string{"health_check", "zpages"},
		"pipelines":  getPipelines(dests),
	}
}

func getPipelines(dests *v1.DestinationList) genericMap {
	traceDests := getDestsForSignal(dests, common.TracesObservabilitySignal)
	metricsDests := getDestsForSignal(dests, common.MetricsObservabilitySignal)
	pipelines := genericMap{}

	if len(traceDests) > 0 {
		pipelines["traces"] = getTracesPipelines(traceDests)
	}

	if len(metricsDests) > 0 {
		pipelines["metrics"] = getMetricsPipelines(metricsDests)
	}

	return pipelines
}

func getDestsForSignal(dests *v1.DestinationList, signal common.ObservabilitySignal) []v1.Destination {
	var destsForSignal []v1.Destination
	for _, dst := range dests.Items {
		for _, s := range dst.Spec.Signals {
			if s == signal {
				destsForSignal = append(destsForSignal, dst)
				break
			}
		}
	}

	return destsForSignal
}

func getTracesPipelines(dests []v1.Destination) genericMap {
	var traceExporters []string
	for _, dst := range dests {
		for e, _ := range getExportersForDest(&dst) {
			traceExporters = append(traceExporters, e)
		}
	}

	return genericMap{
		"receivers":  []string{"otlp", "zipkin"},
		"processors": []string{"batch"},
		"exporters":  traceExporters,
	}
}

func getMetricsPipelines(dests []v1.Destination) genericMap {
	var metricsExporters []string
	for _, dst := range dests {
		for e, _ := range getExportersForDest(&dst) {
			metricsExporters = append(metricsExporters, e)
		}
	}

	return genericMap{
		"receivers":  []string{"otlp"},
		"processors": []string{"batch"},
		"exporters":  metricsExporters,
	}
}
