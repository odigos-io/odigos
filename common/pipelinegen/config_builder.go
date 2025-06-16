package pipelinegen

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
)

func GetGatewayConfig(
	dests []config.ExporterConfigurer,
	processors []config.ProcessorConfigurer,
	memoryLimiterConfig config.GenericMap,
	applySelfTelemetry func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error,
	groupDetails []DataStreams,
) (string, error, *config.ResourceStatuses, []common.ObservabilitySignal) {
	currentConfig := GetBasicConfig(memoryLimiterConfig)
	return CalculateGatewayConfig(currentConfig, dests, processors, applySelfTelemetry, groupDetails)
}

func CalculateGatewayConfig(
	currentConfig *config.Config,
	dests []config.ExporterConfigurer,
	processors []config.ProcessorConfigurer,
	applySelfTelemetry func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error,
	groupDetails []DataStreams,
) (string, error, *config.ResourceStatuses, []common.ObservabilitySignal) {
	configers, err := config.LoadConfigers()
	if err != nil {
		return "", err, nil, nil
	}

	status := &config.ResourceStatuses{
		Destination: make(map[string]error),
		Processor:   make(map[string]error),
	}

	if _, exists := currentConfig.Receivers["otlp"]; !exists {
		return "", fmt.Errorf("missing required receiver 'otlp' on config"), status, nil
	}

	// map of destination ID to list of forward connectors
	// this is used to build the group pipelines
	// e.g. { "destination-1": ["forward/traces/destination-1", "forward/metrics/destination-1", "forward/logs/destination-1"] }
	destForwardConnectors := make(map[string][]string)

	tracesEnabled := false
	metricsEnabled := false
	logsEnabled := false

	// Configure processors
	processorsCfg, tracesProcessors, metricsProcessors, logsProcessors, errs := config.GetCrdProcessorsConfigMap(processors)
	if errs != nil {
		status.Processor = errs
	}
	for processorKey, processorCfg := range processorsCfg {
		currentConfig.Processors[processorKey] = processorCfg
	}

	// TODO: this is a temporary solution to add the small batches processor to the destination pipelines
	// we need to remove this once we have a proper way to processors per pipeline.
	tracesProcessors, smallBatchesEnabled := filterSmallBatchesProcessor(tracesProcessors)

	unifiedDestinationPipelineNames := []string{}
	for _, dest := range dests {
		configer, exists := configers[dest.GetType()]
		if !exists {
			status.Destination[dest.GetID()] = fmt.Errorf("no configer for %s", dest.GetType())
			continue
		}

		destinationPipelineNames, err := configer.ModifyConfig(dest, currentConfig)
		if err != nil {
			status.Destination[dest.GetID()] = err
			continue
		}
		unifiedDestinationPipelineNames = append(unifiedDestinationPipelineNames, destinationPipelineNames...)

		// Create a connector for each destination pipeline [AKA forward connector]
		// Add it as a receiver to the destination pipeline
		for _, pipelineName := range destinationPipelineNames {
			connectorName := "forward/" + pipelineName
			destForwardConnectors[dest.GetID()] = append(destForwardConnectors[dest.GetID()], connectorName)
			currentConfig.Connectors[connectorName] = config.GenericMap{}
			pipeline := currentConfig.Service.Pipelines[pipelineName]
			// add the forward connector as a receiver to the pipeline
			pipeline.Receivers = []string{connectorName}
			// every destination pipeline should have a generic batch processor
			pipeline.Processors = []string{consts.GenericBatchProcessor}

			// track which signals are enabled based on the destination pipeline names
			switch {
			case strings.HasPrefix(pipelineName, "traces/"):
				// relevant only for traces signal
				if smallBatchesEnabled {
					pipeline.Processors = append(pipeline.Processors, consts.SmallBatchesProcessor)
				}
				tracesEnabled = true
			case strings.HasPrefix(pipelineName, "metrics/"):
				metricsEnabled = true
			case strings.HasPrefix(pipelineName, "logs/"):
				logsEnabled = true
			}

			// save the updated pipeline with the new receiver
			currentConfig.Service.Pipelines[pipelineName] = pipeline
		}
	}
	// track which signals are enabled
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

	//  Add pipelines that receive from routing connectors and forward to destinations
	groupPipelines := buildDataStreamPipelines(groupDetails, destForwardConnectors)
	for name, pipe := range groupPipelines {
		currentConfig.Service.Pipelines[name] = pipe
	}

	// Create root pipelines for each signal and connectors
	prepareRootPipelines(currentConfig, groupDetails, tracesProcessors, metricsProcessors, logsProcessors, signals)

	// Optional: Add collector self-observability
	if applySelfTelemetry != nil {
		if err := applySelfTelemetry(currentConfig, unifiedDestinationPipelineNames, GetSignalsRootPipelines()); err != nil {
			return "", err, status, nil
		}
	}

	// Final marshal to YAML
	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return "", err, status, nil
	}

	return string(data), nil, status, signals
}

func prepareRootPipelines(currentConfig *config.Config, groupDetails []DataStreams, tracesProcessors,
	metricsProcessors, logsProcessors []string, signals []common.ObservabilitySignal) {
	// for each signal, create a root pipeline and a connector
	if slices.Contains(signals, common.TracesObservabilitySignal) {
		tracesRootPipelineName := GetTelemetryRootPipeline("traces")
		processors := append([]string{"memory_limiter", "resource/odigos-version"}, tracesProcessors...)

		currentConfig.Service.Pipelines[tracesRootPipelineName] = config.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: processors,
			Exporters:  []string{"odigosrouterconnector/traces"},
		}
		currentConfig.Connectors["odigosrouterconnector/traces"] = config.GenericMap{
			"datastreams": groupDetails,
		}
	}

	if slices.Contains(signals, common.MetricsObservabilitySignal) {
		metricsRootPipelineName := GetTelemetryRootPipeline("metrics")
		processors := append([]string{"memory_limiter", "resource/odigos-version"}, metricsProcessors...)

		currentConfig.Connectors["odigosrouterconnector/metrics"] = config.GenericMap{
			"datastreams": groupDetails,
		}

		currentConfig.Service.Pipelines[metricsRootPipelineName] = config.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: processors,
			Exporters:  []string{"odigosrouterconnector/metrics"},
		}
	}

	if slices.Contains(signals, common.LogsObservabilitySignal) {
		logsRootPipelineName := GetTelemetryRootPipeline("logs")
		processors := append([]string{"memory_limiter", "resource/odigos-version"}, logsProcessors...)

		currentConfig.Connectors["odigosrouterconnector/logs"] = config.GenericMap{
			"datastreams": groupDetails,
		}

		currentConfig.Service.Pipelines[logsRootPipelineName] = config.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: processors,
			Exporters:  []string{"odigosrouterconnector/logs"},
		}
	}
}

func GetBasicConfig(memoryLimiterConfig config.GenericMap) *config.Config {
	return &config.Config{
		Connectors: config.GenericMap{},
		Receivers: config.GenericMap{
			"otlp": config.GenericMap{
				"protocols": config.GenericMap{
					"grpc": config.GenericMap{
						// setting it to a large value to avoid dropping batches.
						"max_recv_msg_size_mib": 128,
						"endpoint":              "0.0.0.0:4317",
						// The Node Collector opens a gRPC stream to send data.
						// This ensures that the Node Collector establishes a new connection when the Gateway scales up
						// to include additional instances.
						"keepalive": config.GenericMap{
							"server_parameters": config.GenericMap{
								"max_connection_age":       consts.GatewayMaxConnectionAge,
								"max_connection_age_grace": consts.GatewayMaxConnectionAgeGrace,
							},
						},
					},
					// Node collectors send in gRPC, so this is probably not needed
					"http": config.GenericMap{
						"endpoint": "0.0.0.0:4318",
					},
				},
			},
		},
		Processors: config.GenericMap{
			"memory_limiter": memoryLimiterConfig,
			"resource/odigos-version": config.GenericMap{
				"attributes": []config.GenericMap{
					{
						"key":    "odigos.version",
						"value":  "${ODIGOS_VERSION}",
						"action": "upsert",
					},
				},
			},
			"batch/generic-batch-processor": config.GenericMap{}, // Currently configured with default values
		},
		Extensions: config.GenericMap{
			"health_check": config.GenericMap{
				"endpoint": "0.0.0.0:13133",
			},
			"pprof": config.GenericMap{
				"endpoint": "0.0.0.0:1777",
			},
		},
		Exporters: map[string]interface{}{},
		Service: config.Service{
			Pipelines:  map[string]config.Pipeline{},
			Extensions: []string{"health_check", "pprof"},
		},
	}
}

func filterSmallBatchesProcessor(tracesProcessors []string) ([]string, bool) {
	smallBatchesEnabled := false
	var filtered []string

	for _, processor := range tracesProcessors {
		if processor == consts.SmallBatchesProcessor {
			smallBatchesEnabled = true
			continue // skip adding it to filtered slice
		}
		filtered = append(filtered, processor)
	}

	return filtered, smallBatchesEnabled
}
