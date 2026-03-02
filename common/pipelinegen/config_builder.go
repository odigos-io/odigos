package pipelinegen

import (
	"fmt"
	"slices"
	"strings"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"gopkg.in/yaml.v2"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
)

type GatewayConfigOptions struct {
	ServiceGraphDisabled  *bool
	ClusterMetricsEnabled *bool
	OdigosNamespace       string

	// Sampling config option
	SamplingEnabled              *bool
	TraceAggregationWaitDuration *string
}

func GetGatewayConfig(
	dests []config.ExporterConfigurer,
	processors []config.ProcessorConfigurer,
	applySelfTelemetry func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error,
	dataStreamsDetails []DataStreams,
	gatewayOptions GatewayConfigOptions,
) (string, error, *config.ResourceStatuses, []common.ObservabilitySignal) {
	currentConfig := GetBasicConfig()
	return CalculateGatewayConfig(currentConfig, dests, processors, applySelfTelemetry, dataStreamsDetails, gatewayOptions)
}

//nolint:funlen // This function handles complex gateway configuration logic that is difficult to break down further
func CalculateGatewayConfig(
	currentConfig *config.Config,
	dests []config.ExporterConfigurer,
	processors []config.ProcessorConfigurer,
	applySelfTelemetry func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error,
	dataStreamsDetails []DataStreams,
	gatewayOptions GatewayConfigOptions,
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
	processorsResults := config.CrdProcessorToConfig(processors)
	if len(processorsResults.Errs) != 0 {
		for processorKey, err := range processorsResults.Errs {
			status.Processor[processorKey] = err
		}
	}
	for processorKey, processorCfg := range processorsResults.ProcessorsConfig.Processors {
		currentConfig.Processors[processorKey] = processorCfg
	}

	// If sampling v2 is enabled, we need to add the groupbytrace processor to the traces processors.
	if gatewayOptions.SamplingEnabled != nil && *gatewayOptions.SamplingEnabled {
		groupbytraceProcessor := config.GenericMap{
			"wait_duration": gatewayOptions.TraceAggregationWaitDuration,
		}
		currentConfig.Processors[consts.GroupByTraceProcessorV2] = groupbytraceProcessor
		// add the groupbytrace processor to the beginning of the traces processors
		processorsResults.TracesProcessors = append([]string{consts.GroupByTraceProcessorV2}, processorsResults.TracesProcessors...)
	}

	allTracesProcessors := make([]string, 0, len(processorsResults.TracesProcessors)+len(processorsResults.TracesProcessorsPostSpanMetrics))
	allTracesProcessors = append(allTracesProcessors, processorsResults.TracesProcessors...)
	allTracesProcessors = append(allTracesProcessors, processorsResults.TracesProcessorsPostSpanMetrics...)

	// TODO: this is a temporary solution to add the small batches processor to the destination pipelines
	// we need to remove this once we have a proper way to processors per pipeline.
	allTracesProcessors, smallBatchesEnabled := filterSmallBatchesProcessor(allTracesProcessors)

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
			pipeline.Receivers = append(pipeline.Receivers, connectorName)
			// every destination pipeline should have a generic batch processor
			pipeline.Processors = append(pipeline.Processors, consts.GenericBatchProcessorConfigKey)

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

		status.Destination[dest.GetID()] = nil // mark this destination as success
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
	insertRootPipelinesToConfig(currentConfig,
		dataStreamsDetails,
		allTracesProcessors,
		processorsResults.MetricsProcessors,
		processorsResults.LogsProcessors,
		enabledSignals)

	// Optional: Add collector self-observability
	if applySelfTelemetry != nil {
		if err := applySelfTelemetry(currentConfig, unifiedDestinationPipelineNames, GetSignalsRootPipelineNames()); err != nil {
			return "", err, status, nil
		}
	}

	// Defensive nil-checks to avoid panic on optional *bool fields.
	// Defaults:
	// - ServiceGraphDisabled: assume false (enabled) if nil
	// - ClusterMetricsEnabled: assume false (disabled) if nil
	if tracesEnabled && (gatewayOptions.ServiceGraphDisabled == nil || !*gatewayOptions.ServiceGraphDisabled) {
		insertServiceGraphPipeline(currentConfig)
	}
	if metricsEnabled && gatewayOptions.ClusterMetricsEnabled != nil && *gatewayOptions.ClusterMetricsEnabled {
		insertClusterMetricsResources(currentConfig, gatewayOptions.OdigosNamespace)
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
	rootPipelineName := GetTelemetryRootPipelineName(signal)
	fullProcessors := append([]string{"resource/odigos-version"}, processors...)

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

func insertServiceGraphPipeline(currentConfig *config.Config) {
	// Add the service graph exporter to expose the service graph metrics to prometheus
	currentConfig.Exporters["prometheus/servicegraph"] = config.GenericMap{
		"endpoint":  fmt.Sprintf("localhost:%d", consts.ServiceGraphEndpointPort),
		"namespace": "servicegraph",
	}

	// adding the service graph scrape config to prometheus/self-metrics receiver
	err := AddServiceGraphScrapeConfig(currentConfig)
	if err != nil {
		return
	}

	// Add the service graph connector to receive the service graph metrics from the root traces pipeline
	// Retain incomplete edges for up to 15s to allow delayed span matching
	// Clean up every 5s to reduce memory pressure and avoid stale edges
	currentConfig.Connectors[consts.ServiceGraphConnectorName] = config.GenericMap{
		"store": config.GenericMap{
			"ttl": "15s",
		},
		"store_expiration_loop": "5s",
		"dimensions":            []string{string(semconv.ServiceNameKey)},
	}

	// Add the service graph pipeline to receive the service graph metrics from the root traces pipeline
	currentConfig.Service.Pipelines["metrics/servicegraph"] = config.Pipeline{
		Receivers: []string{consts.ServiceGraphConnectorName},
		Exporters: []string{"prometheus/servicegraph"},
	}

	// Add the service graph exporter to the root traces pipeline
	rootPipelineName := GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	// This pipeline should already exist because entering this function means that traces are enabled, but we'll check just in case
	pipeline, exists := currentConfig.Service.Pipelines[rootPipelineName]
	if !exists {
		return
	}
	pipeline.Exporters = append(pipeline.Exporters, consts.ServiceGraphConnectorName)
	currentConfig.Service.Pipelines[rootPipelineName] = pipeline
}

func GetBasicConfig() *config.Config {
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
			"resource/odigos-version": config.GenericMap{
				"attributes": []config.GenericMap{
					{
						"key":    "odigos.version",
						"value":  "${ODIGOS_VERSION}",
						"action": "upsert",
					},
				},
			},
			consts.GenericBatchProcessorConfigKey: config.GenericMap{},
		},
		Extensions: config.GenericMap{
			"health_check": config.GenericMap{
				"endpoint": "0.0.0.0:13133",
			},
			"pprof": config.GenericMap{
				"endpoint": "0.0.0.0:1777",
			},
			consts.OdigosWorkloadConfigExtensionName: config.GenericMap{},
		},
		Exporters: map[string]interface{}{},
		Service: config.Service{
			Pipelines:  map[string]config.Pipeline{},
			Extensions: []string{"health_check", "pprof", consts.OdigosWorkloadConfigExtensionName},
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
func AddServiceGraphScrapeConfig(c *config.Config) error {
	servicegraphScrape := config.GenericMap{
		"job_name":        consts.ServiceGraphConnectorName,
		"scrape_interval": "10s",
		"static_configs": []config.GenericMap{
			{
				"targets": []string{fmt.Sprintf("127.0.0.1:%d", consts.ServiceGraphEndpointPort)},
			},
		},
		"metric_relabel_configs": []config.GenericMap{
			{
				"source_labels": []string{"__name__"},
				"regex":         "^servicegraph_traces_service_graph_request_total$",
				"action":        "keep",
			},
		},
	}

	receiverVal, ok := c.Receivers["prometheus/self-metrics"]
	if !ok {
		return fmt.Errorf("receiver config is not a map")
	}
	receiver, ok := receiverVal.(config.GenericMap)
	if !ok {
		return fmt.Errorf("receiver config is not a map")
	}

	configMap, ok := receiver["config"].(config.GenericMap)
	if !ok {
		return fmt.Errorf("scrape configs is not a list")
	}

	scrapeConfigs, ok := configMap["scrape_configs"].([]config.GenericMap)
	if !ok {
		return fmt.Errorf("scrape configs is not a list")
	}

	// Append new servicegraph scrape config
	scrapeConfigs = append(scrapeConfigs, servicegraphScrape)

	// Reassign
	configMap["scrape_configs"] = scrapeConfigs
	receiver["config"] = configMap
	c.Receivers["prometheus/self-metrics"] = receiver
	return nil
}

func insertClusterMetricsResources(currentConfig *config.Config, odigosNs string) {
	// setup the leader elector extension, this is to avoid all gateways instances to scrape the cluster metrics at the same time
	currentConfig.Extensions["k8s_leader_elector"] = config.GenericMap{
		"lease_name":      "odigos-gateway-leader",
		"lease_namespace": odigosNs,
	}

	// add this to the service.extensions
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, "k8s_leader_elector")

	// setup the k8s_cluster_receiver
	currentConfig.Receivers["k8s_cluster"] = config.GenericMap{
		"k8s_leader_elector": "k8s_leader_elector",
	}

	// Add the cluster metrics resources to the root metrics pipeline
	rootPipelineName := GetTelemetryRootPipelineName(common.MetricsObservabilitySignal)
	pipeline, exists := currentConfig.Service.Pipelines[rootPipelineName]
	if !exists {
		return
	}

	pipeline.Receivers = append(pipeline.Receivers, "k8s_cluster")
	currentConfig.Service.Pipelines[rootPipelineName] = pipeline
}
