package config

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	memoryLimiterProcessorName = "memory_limiter"
)

var availableConfigers = []Configer{&Middleware{}, &Honeycomb{}, &GrafanaCloudPrometheus{}, &GrafanaCloudTempo{}, &GrafanaCloudLoki{}, &Datadog{}, &NewRelic{}, &Logzio{}, &Prometheus{},
	&Tempo{}, &Loki{}, &Jaeger{}, &GenericOTLP{}, &OTLPHttp{}, &Elasticsearch{}, &Quickwit{}, &Signoz{}, &Qryn{},
	&OpsVerse{}, &Splunk{}, &Lightstep{}, &GoogleCloud{}, &GoogleCloudStorage{}, &Sentry{}, &AzureBlobStorage{},
	&AWSS3{}, &Dynatrace{}, &Chronosphere{}, &ElasticAPM{}, &Axiom{}, &SumoLogic{}, &Coralogix{}}

type Configer interface {
	DestType() common.DestinationType
	ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error
}

func Calculate(dests []common.ExporterConfigurer, processors []*odigosv1.Processor, memoryLimiterConfig commonconf.GenericMap) (string, error, map[string]error) {
	currentConfig := getBasicConfig(memoryLimiterConfig)

	configers, err := loadConfigers()
	if err != nil {
		return "", err, nil
	}

	destsStatus := map[string]error{}

	for _, dest := range dests {
		configer, exists := configers[dest.GetType()]
		if !exists {
			return "", fmt.Errorf("no configer for %s", dest.GetType()), nil
		}

		err := configer.ModifyConfig(dest, currentConfig)
		destsStatus[dest.GetName()] = err
	}

	processorsCfg, tracesProcessors, metricsProcessors, logsProcessors := commonconf.GetCrdProcessorsConfigMap(processors)
	for processorKey, processorCfg := range processorsCfg {
		currentConfig.Processors[processorKey] = processorCfg
	}

	for pipelineName, pipeline := range currentConfig.Service.Pipelines {
		if strings.HasPrefix(pipelineName, "traces/") {
			pipeline.Processors = append(tracesProcessors, pipeline.Processors...)
		} else if strings.HasPrefix(pipelineName, "metrics/") {
			pipeline.Processors = append(metricsProcessors, pipeline.Processors...)
		} else if strings.HasPrefix(pipelineName, "logs/") {
			pipeline.Processors = append(logsProcessors, pipeline.Processors...)
		}

		// basic config common to all pipelines
		pipeline.Receivers = []string{"otlp"}
		// memory limiter processor should be the first processor in the pipeline
		pipeline.Processors = append([]string{memoryLimiterProcessorName, "batch", "resource/odigos-version"}, pipeline.Processors...)
		currentConfig.Service.Pipelines[pipelineName] = pipeline
	}

	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return "", err, destsStatus
	}

	return string(data), nil, destsStatus
}

func getBasicConfig(memoryLimiterConfig commonconf.GenericMap) *commonconf.Config {
	empty := struct{}{}
	return &commonconf.Config{
		Receivers: commonconf.GenericMap{
			"otlp": commonconf.GenericMap{
				"protocols": commonconf.GenericMap{
					"grpc": commonconf.GenericMap{
						// setting it to a large value to avoid dropping batches.
						"max_recv_msg_size_mib": 128 * 1024 * 1024,
					},
					"http": empty,
				},
			},
		},
		Processors: commonconf.GenericMap{
			memoryLimiterProcessorName: memoryLimiterConfig,
			"batch":                    empty,
			"resource/odigos-version": commonconf.GenericMap{
				"attributes": []commonconf.GenericMap{
					{
						"key":    "odigos.version",
						"value":  "${ODIGOS_VERSION}",
						"action": "upsert",
					},
				},
			},
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

func isSignalExists(dest common.ExporterConfigurer, signal common.ObservabilitySignal) bool {
	for _, s := range dest.GetSignals() {
		if s == signal {
			return true
		}
	}

	return false
}

func isTracingEnabled(dest common.ExporterConfigurer) bool {
	return isSignalExists(dest, common.TracesObservabilitySignal)
}

func isMetricsEnabled(dest common.ExporterConfigurer) bool {
	return isSignalExists(dest, common.MetricsObservabilitySignal)
}

func isLoggingEnabled(dest common.ExporterConfigurer) bool {
	return isSignalExists(dest, common.LogsObservabilitySignal)
}

func addProtocol(s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}

	return fmt.Sprintf("http://%s", s)
}
