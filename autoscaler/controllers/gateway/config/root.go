package config

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

var availableConfigers = []Configer{&Middleware{}, &Honeycomb{}, &Grafana{}, &Datadog{}, &NewRelic{}, &Logzio{}, &Prometheus{},
	&Tempo{}, &Loki{}, &Jaeger{}, &GenericOTLP{}, &Elasticsearch{}, &Signoz{}, &Qryn{},
	&OpsVerse{}, &Splunk{}, &Lightstep{}, &GoogleCloud{}, &GoogleCloudStorage{}, &Sentry{}, &AzureBlobStorage{},
	&AWSS3{}, &Dynatrace{}, &Chronosphere{}, &ElasticAPM{}, &Axiom{}}

type Configer interface {
	DestType() common.DestinationType
	ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config)
}

func Calculate(dests *odigosv1.DestinationList) (string, error) {
	currentConfig := getBasicConfig()
	configers, err := loadConfigers()
	if err != nil {
		return "", err
	}

	for _, dest := range dests.Items {
		configer, exists := configers[dest.Spec.Type]
		if !exists {
			return "", fmt.Errorf("no configer for %s", dest.Spec.Type)
		}

		configer.ModifyConfig(&dest, currentConfig)
	}

	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getBasicConfig() *commonconf.Config {
	empty := struct{}{}
	return &commonconf.Config{
		Receivers: commonconf.GenericMap{
			"otlp": commonconf.GenericMap{
				"protocols": commonconf.GenericMap{
					"grpc": empty,
					"http": empty,
				},
			},
		},
		Processors: commonconf.GenericMap{
			"batch": empty,
		},
		Extensions: commonconf.GenericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Exporters: map[string]interface{}{},
		Service: commonconf.Service{
			Pipelines:  map[string]commonconf.Pipeline{},
			Extensions: []string{"health_check", "zpages"},
		},
	}
}

func loadConfigers() (map[common.DestinationType]Configer, error) {
	configers := map[common.DestinationType]Configer{}
	for _, configer := range availableConfigers {
		if _, exists := configers[configer.DestType()]; exists {
			return nil, fmt.Errorf("duplicate configer for %s", configer.DestType())
		}

		configers[configer.DestType()] = configer
	}

	return configers, nil
}

func isSignalExists(dest *odigosv1.Destination, signal common.ObservabilitySignal) bool {
	for _, s := range dest.Spec.Signals {
		if s == signal {
			return true
		}
	}

	return false
}

func isTracingEnabled(dest *odigosv1.Destination) bool {
	return isSignalExists(dest, common.TracesObservabilitySignal)
}

func isMetricsEnabled(dest *odigosv1.Destination) bool {
	return isSignalExists(dest, common.MetricsObservabilitySignal)
}

func isLoggingEnabled(dest *odigosv1.Destination) bool {
	return isSignalExists(dest, common.LogsObservabilitySignal)
}

func addProtocol(s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}

	return fmt.Sprintf("http://%s", s)
}
