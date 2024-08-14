package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/odigos-io/odigos/common"
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

func Calculate(dests []ExporterConfigurer, processors []ProcessorConfigurer, memoryLimiterConfig GenericMap, applySelfTelemetry func(c *Config) error) (string, error, *ResourceStatuses, []common.ObservabilitySignal) {
	currentConfig, prefixProcessors := getBasicConfig(memoryLimiterConfig)
	return CalculateWithBase(currentConfig, prefixProcessors, dests, processors, applySelfTelemetry)
}

func CalculateWithBase(currentConfig *Config, prefixProcessors []string, dests []ExporterConfigurer, processors []ProcessorConfigurer, applySelfTelemetry func(c *Config) error) (string, error, *ResourceStatuses, []common.ObservabilitySignal) {
	configers, err := LoadConfigers()
	if err != nil {
		return "", err, nil, nil
	}

	status := &ResourceStatuses{
		Destination: make(map[string]error),
		Processor:   make(map[string]error),
	}

	for _, p := range prefixProcessors {
		_, exists := currentConfig.Processors[p]
		if !exists {
			return "", fmt.Errorf("missing prefix processor '%s' on config", p), status, nil
		}
	}

	if _, exists := currentConfig.Receivers["otlp"]; !exists {
		return "", fmt.Errorf("missing required receiver 'otlp' on config"), status, nil
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

	tracesEnabled := false
	metricsEnabled := false
	logsEnabled := false

	for pipelineName, pipeline := range currentConfig.Service.Pipelines {
		if strings.Contains(pipelineName, "otelcol") {
			continue
		}
		if strings.HasPrefix(pipelineName, "traces/") {
			pipeline.Processors = append(tracesProcessors, pipeline.Processors...)
			tracesEnabled = true
		} else if strings.HasPrefix(pipelineName, "metrics/") {
			pipeline.Processors = append(metricsProcessors, pipeline.Processors...)
			metricsEnabled = true
		} else if strings.HasPrefix(pipelineName, "logs/") {
			pipeline.Processors = append(logsProcessors, pipeline.Processors...)
			logsEnabled = true
		}

		// basic config common to all pipelines
		pipeline.Receivers = append([]string{"otlp"}, pipeline.Receivers...)
		// memory limiter processor should be the first processor in the pipeline
		// odigostrafficmetrics processor should be the last processor in the pipeline
		pipeline.Processors = slices.Concat(prefixProcessors, pipeline.Processors)
		currentConfig.Service.Pipelines[pipelineName] = pipeline
	}

	// Apply self telemetry to the configuration
	// It is important to apply this after the main pipelines are created, since this operation will add a metrics pipeline
	// which is responsible for collecting metrics about the collector itself.
	if applySelfTelemetry != nil {
		err := applySelfTelemetry(currentConfig)
		if err != nil {
			return "", err, status, nil
		}
	}

	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return "", err, status, nil
	}

	signals := []common.ObservabilitySignal{}
	if tracesEnabled {
		signals = append(signals, common.TracesObservabilitySignal)
	}
	if metricsEnabled {
		signals = append(signals, common.MetricsObservabilitySignal)
	}
	if logsEnabled {
		signals = append(signals, common.LogsObservabilitySignal)
	}

	return string(data), nil, status, signals
}

// getBasicConfig returns a basic configuration for the cluster collector.
// It includes the basic receivers, processors, exporters, extensions, and service configuration.
// In addition it returns prefix processors that should be added to beginning of each pipeline.
func getBasicConfig(memoryLimiterConfig GenericMap) (*Config, []string) {
	return &Config{
		Receivers: GenericMap{
			"otlp": GenericMap{
				"protocols": GenericMap{
					"grpc": GenericMap{
						// setting it to a large value to avoid dropping batches.
						"max_recv_msg_size_mib": 128 * 1024 * 1024,
						"endpoint": "0.0.0.0:4317",
					},
					// Node collectors send in gRPC, so this is probably not needed
					"http": GenericMap{
						"endpoint": "0.0.0.0:4318",
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
		},
		Extensions: GenericMap{
			"health_check": GenericMap{
				"endpoint": "0.0.0.0:13133",
			},
		},
		Exporters:  map[string]interface{}{},
		Connectors: map[string]interface{}{},
		Service: Service{
			Pipelines:  map[string]Pipeline{},
			Extensions: []string{"health_check"},
		},
	},
	[]string{memoryLimiterProcessorName, "resource/odigos-version"}
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
