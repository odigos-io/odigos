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
	dataStreamsDetails []DataStreams,
) (string, error, *config.ResourceStatuses, []common.ObservabilitySignal) {
	currentConfig := GetBasicConfig(memoryLimiterConfig)
	return CalculateGatewayConfig(currentConfig, dests, processors, applySelfTelemetry, dataStreamsDetails)
}

func CalculateGatewayConfig(
	currentConfig *config.Config,
	dests []config.ExporterConfigurer,
	processors []config.ProcessorConfigurer,
	applySelfTelemetry func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error,
	dataStreamsDetails []DataStreams,
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
	// this is used to build the data stream pipelines
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
			pipeline.Processors = []string{consts.GenericBatchProcessorConfigKey}

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
	enabledSignals := []common.ObservabilitySignal{}

	if tracesEnabled {
		enabledSignals = append(enabledSignals, common.TracesObservabilitySignal)
	}
	if metricsEnabled {
		enabledSignals = append(enabledSignals, common.MetricsObservabilitySignal)
	}
	if logsEnabled {
		enabledSignals = append(enabledSignals, common.LogsObservabilitySignal)
	}

	//  Add pipelines that receive from routing connectors and forward to destinations
	dataStreamPipelines := buildDataStreamPipelines(dataStreamsDetails, destForwardConnectors)
	for name, pipe := range dataStreamPipelines {
		currentConfig.Service.Pipelines[name] = pipe
	}

	// Create root pipelines for each signal and connectors
	insertRootPipelinesToConfig(currentConfig, dataStreamsDetails, tracesProcessors, metricsProcessors, logsProcessors, enabledSignals)

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

	return string(data), nil, status, enabledSignals
}

func insertRootPipelinesToConfig(currentConfig *config.Config, dataStreamsDetails []DataStreams,
	tracesProcessors, metricsProcessors, logsProcessors []string, signals []common.ObservabilitySignal) {
	if slices.Contains(signals, common.TracesObservabilitySignal) {
		applyRootPipelineForSignal(
			currentConfig,
			common.TracesObservabilitySignal,
			tracesProcessors,
			dataStreamsDetails,
		)
	}

	if slices.Contains(signals, common.MetricsObservabilitySignal) {
		applyRootPipelineForSignal(
			currentConfig,
			common.MetricsObservabilitySignal,
			metricsProcessors,
			dataStreamsDetails,
		)
	}

	if slices.Contains(signals, common.LogsObservabilitySignal) {
		applyRootPipelineForSignal(
			currentConfig,
			common.LogsObservabilitySignal,
			logsProcessors,
			dataStreamsDetails,
		)
	}
}

func applyRootPipelineForSignal(currentConfig *config.Config, signal common.ObservabilitySignal,
	processors []string, dataStreamsDetails []DataStreams) {
	rootPipelineName := GetTelemetryRootPipeline(signal)
	fullProcessors := append([]string{"memory_limiter", "resource/odigos-version"}, processors...)

	connectorName := fmt.Sprintf("odigosrouterconnector/%s", strings.ToLower(string(signal)))
	currentConfig.Connectors[connectorName] = config.GenericMap{
		"datastreams": dataStreamsDetails,
	}

	currentConfig.Service.Pipelines[rootPipelineName] = config.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: fullProcessors,
		Exporters:  []string{connectorName},
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
			consts.GenericBatchProcessorConfigKey: config.GenericMap{}, // Currently configured with default values
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
	filtered := make([]string, 0, len(tracesProcessors))

	for _, processor := range tracesProcessors {
		if processor == consts.SmallBatchesProcessor {
			smallBatchesEnabled = true
			continue // skip adding it to filtered slice
		}
		filtered = append(filtered, processor)
	}

	return filtered, smallBatchesEnabled
}
