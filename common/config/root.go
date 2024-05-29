package config

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/odigos-io/odigos/common"
)

const (
	memoryLimiterProcessorName = "memory_limiter"
)

var availableConfigers = []Configer{&Middleware{}, &Honeycomb{}, &GrafanaCloudPrometheus{}, &GrafanaCloudTempo{}, &GrafanaCloudLoki{}, &Datadog{}, &NewRelic{}, &Logzio{}, &Prometheus{},
	&Tempo{}, &Loki{}, &Jaeger{}, &GenericOTLP{}, &OTLPHttp{}, &Elasticsearch{}, &Quickwit{}, &Signoz{}, &Qryn{},
	&OpsVerse{}, &Splunk{}, &Lightstep{}, &GoogleCloud{}, &GoogleCloudStorage{}, &Sentry{}, &AzureBlobStorage{},
	&AWSS3{}, &Dynatrace{}, &Chronosphere{}, &ElasticAPM{}, &Axiom{}, &SumoLogic{}, &Coralogix{}, &Causely{}}

type Configer interface {
	DestType() common.DestinationType
	ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error
}

type ResourceStatuses struct {
	Destination map[string]error
	Processor   map[string]error
}

func Calculate(dests []ExporterConfigurer, processors []ProcessorConfigurer, memoryLimiterConfig GenericMap) (string, error, *ResourceStatuses) {
	currentConfig := getBasicConfig(memoryLimiterConfig)

	configers, err := loadConfigers()
	if err != nil {
		return "", err, nil
	}

	status := &ResourceStatuses{
		Destination: make(map[string]error),
		Processor:   make(map[string]error),
	}

	for _, dest := range dests {
		configer, exists := configers[dest.GetType()]
		if !exists {
			return "", fmt.Errorf("no configer for %s", dest.GetType()), nil
		}

		err := configer.ModifyConfig(dest, currentConfig)
		status.Destination[dest.GetName()] = err
	}

	processorsCfg, tracesProcessors, metricsProcessors, logsProcessors, errs := GetCrdProcessorsConfigMap(processors)
	if errs != nil {
		status.Processor = errs
	}
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
		pipeline.Receivers = append([]string{"otlp"}, pipeline.Receivers...)
		// memory limiter processor should be the first processor in the pipeline
		pipeline.Processors = append([]string{memoryLimiterProcessorName, "batch", "resource/odigos-version"}, pipeline.Processors...)
		currentConfig.Service.Pipelines[pipelineName] = pipeline
	}

	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return "", err, status
	}

	return string(data), nil, status
}

func getBasicConfig(memoryLimiterConfig GenericMap) *Config {
	empty := struct{}{}
	return &Config{
		Receivers: GenericMap{
			"otlp": GenericMap{
				"protocols": GenericMap{
					"grpc": GenericMap{
						// setting it to a large value to avoid dropping batches.
						"max_recv_msg_size_mib": 128 * 1024 * 1024,
					},
					"http": empty,
				},
			},
		},
		Processors: GenericMap{
			memoryLimiterProcessorName: memoryLimiterConfig,
			"batch":                    empty,
			"resource/odigos-version": GenericMap{
				"attributes": []GenericMap{
					{
						"key":    "odigos.version",
						"value":  "${ODIGOS_VERSION}",
						"action": "upsert",
					},
				},
			},
		},
		Extensions: GenericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Exporters:  map[string]interface{}{},
		Connectors: map[string]interface{}{},
		Service: Service{
			Pipelines:  map[string]Pipeline{},
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

func isSignalExists(dest SignalSpecific, signal common.ObservabilitySignal) bool {
	for _, s := range dest.GetSignals() {
		if s == signal {
			return true
		}
	}

	return false
}

func isTracingEnabled(dest SignalSpecific) bool {
	return isSignalExists(dest, common.TracesObservabilitySignal)
}

func isMetricsEnabled(dest SignalSpecific) bool {
	return isSignalExists(dest, common.MetricsObservabilitySignal)
}

func isLoggingEnabled(dest SignalSpecific) bool {
	return isSignalExists(dest, common.LogsObservabilitySignal)
}

func addProtocol(s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}

	return fmt.Sprintf("http://%s", s)
}
