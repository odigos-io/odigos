package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
)

const (
	memoryLimiterProcessorName = "memory_limiter"
)

var availableConfigers = []Configer{
	&Middleware{}, &Honeycomb{}, &GrafanaCloudPrometheus{}, &GrafanaCloudTempo{},
	&GrafanaCloudLoki{}, &Datadog{}, &NewRelic{}, &Logzio{}, &Prometheus{},
	&Tempo{}, &Loki{}, &Jaeger{}, &GenericOTLP{}, &OTLPHttp{}, &Elasticsearch{}, &Quickwit{}, &Signoz{}, &Qryn{},
	&OpsVerse{}, &Splunk{}, &Lightstep{}, &GoogleCloud{}, &GoogleCloudStorage{}, &Sentry{}, &AzureBlobStorage{},
	&AWSS3{}, &Dynatrace{}, &Chronosphere{}, &ElasticAPM{}, &Axiom{}, &SumoLogic{}, &Coralogix{}, &Clickhouse{},
	&Causely{}, &Uptrace{}, &Debug{},
}

type Configer interface {
	DestType() common.DestinationType
	ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error
}

type ResourceStatuses struct {
	Destination map[string]error
	Processor   map[string]error
}

func Calculate(dests []ExporterConfigurer, processors []ProcessorConfigurer, memoryLimiterConfig GenericMap) (string, error, *ResourceStatuses) {
	currentConfig, prefixProcessors, suffixProcessors := getBasicConfig(memoryLimiterConfig)
	return CalculateWithBase(currentConfig, prefixProcessors, suffixProcessors, dests, processors)
}

func CalculateWithBase(currentConfig *Config, prefixProcessors []string, suffixProcessors []string, dests []ExporterConfigurer, processors []ProcessorConfigurer) (string, error, *ResourceStatuses) {
	configers, err := LoadConfigers()
	if err != nil {
		return "", err, nil
	}

	status := &ResourceStatuses{
		Destination: make(map[string]error),
		Processor:   make(map[string]error),
	}

	for _, p := range prefixProcessors {
		_, exists := currentConfig.Processors[p]
		if !exists {
			return "", fmt.Errorf("missing prefix processor '%s' on config", p), status
		}
	}

	for _, s := range suffixProcessors {
		_, exists := currentConfig.Processors[s]
		if !exists {
			return "", fmt.Errorf("missing suffix processor '%s' on config", s), status
		}
	}

	if _, exists := currentConfig.Receivers["otlp"]; !exists {
		return "", fmt.Errorf("missing required receiver 'otlp' on config"), status
	}

	for _, dest := range dests {
		configer, exists := configers[dest.GetType()]
		if !exists {
			status.Destination[dest.GetID()] = fmt.Errorf("no configer for %s", dest.GetType())
			continue
		}

		err := configer.ModifyConfig(dest, currentConfig)
		status.Destination[dest.GetID()] = err

		// If configurer ran without errors, but there were no signals enabled, warn the user
		if len(dest.GetSignals()) == 0 && err == nil {
			status.Destination[dest.GetID()] = fmt.Errorf("no signals enabled for %s(%s)", dest.GetID(), dest.GetType())
		}
	}

	processorsCfg, tracesProcessors, metricsProcessors, logsProcessors, errs := GetCrdProcessorsConfigMap(processors)
	if errs != nil {
		status.Processor = errs
	}
	for processorKey, processorCfg := range processorsCfg {
		currentConfig.Processors[processorKey] = processorCfg
	}

	for pipelineName, pipeline := range currentConfig.Service.Pipelines {
		if strings.Contains(pipelineName, "otelcol") {
			continue
		}
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
		// odigostrafficmetrics processor should be the last processor in the pipeline
		pipeline.Processors = slices.Concat(prefixProcessors, pipeline.Processors, suffixProcessors)
		currentConfig.Service.Pipelines[pipelineName] = pipeline
	}

	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return "", err, status
	}

	return string(data), nil, status
}

// getBasicConfig returns a basic configuration for the cluster collector.
// It includes the basic receivers, processors, exporters, extensions, and service configuration.
// In addition it returns prefix and suffix processors that should be added to beginning and end of each pipeline.
func getBasicConfig(memoryLimiterConfig GenericMap) (*Config, []string, []string) {
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
			"prometheus": GenericMap{
				"config": GenericMap{
					"scrape_configs": []GenericMap{
						{
							"job_name": "otelcol",
							"scrape_interval": "10s",
							"static_configs": []GenericMap{
								{
									"targets": []string{"127.0.0.1:8888"},
								},
							},
							"metric_relabel_configs": []GenericMap{
								{
									"source_labels": []string{"__name__"},
									"regex": "(.*odigos.*|^otelcol_processor_accepted.*|^otelcol_exporter_sent.*)",
									"action": "keep",
								},
							},
						},
					},
				},
			},
		},
		Processors: GenericMap{
			memoryLimiterProcessorName: memoryLimiterConfig,
			"resource/odigos-version": GenericMap{
				"attributes": []GenericMap{
					{
						"key":    "odigos.version",
						"value":  "${ODIGOS_VERSION}",
						"action": "upsert",
					},
				},
			},
			// odigostrafficmetrics processor should be the last processor in the pipeline
			// as it helps to calculate the size of the data being exported.
			// In case of performance impact caused by this processor, we should modify this config to reduce the sampling ratio.
			"odigostrafficmetrics": empty,
		},
		Extensions: GenericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Exporters:  map[string]interface{}{
			"otlp/ui": GenericMap{
				"endpoint": fmt.Sprintf("ui.%s:%d", env.GetCurrentNamespace(), consts.OTLPPort),
				"tls": GenericMap{
					"insecure": true,
				},
				"headers": GenericMap{
					k8sconsts.OdigosPodNameHeaderKey: "${POD_NAME}",
				},
			},
		},
		Connectors: map[string]interface{}{},
		Service: Service{
			Pipelines:  map[string]Pipeline{
				"metrics/otelcol": {
					Receivers: []string{"prometheus"},
					Exporters: []string{"otlp/ui"},
				},
			},
			Extensions: []string{"health_check", "zpages"},
			Telemetry: Telemetry{
				Metrics: GenericMap{
					"address": "0.0.0.0:8888",
				},
			},
		},
	},
	[]string{memoryLimiterProcessorName, "resource/odigos-version"}, []string{"odigostrafficmetrics"}
}

func LoadConfigers() (map[common.DestinationType]Configer, error) {
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
