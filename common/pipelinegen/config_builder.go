package pipelinegen

import (
	"fmt"
	"slices"
	"strings"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"gopkg.in/yaml.v2"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
)

type GatewayConfigOptions struct {
	ServiceGraph          common.ServiceGraphOptions
	ClusterMetricsEnabled *bool
	OdigosNamespace       string

	// the name of the extension for accessing odigos configurations.
	// the extension and it's name are platform specific.
	OdigosConfigExtensionName *string

	// groupbytrace wait duration when tail sampling or service I/O trace correlations are active.
	TraceAggregationWaitDuration *string

	// Tail sampling v2 processors when tail sampling is active.
	TailSamplingEnabled    *bool
	SamplingDryRun         bool
	SamplingSpanAttributes *sampling.SpanSamplingAttributesConfiguration

	// Trace correlations configuration for the serviceio connector (service I/O metrics).
	TraceCorrelationsServiceIO *common.TraceCorrelationsServiceIOConfiguration
}

func GetGatewayConfig(
	dests []config.ExporterConfigurer,
	processors []config.ProcessorConfigurer,
	applySelfTelemetry func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error,
	dataStreamsDetails []DataStreams,
	gatewayOptions *GatewayConfigOptions,
) (string, error, *config.ResourceStatuses, []common.ObservabilitySignal) {
	cfg, err, status, signals := CalculateGatewayConfig(dests, processors, applySelfTelemetry, dataStreamsDetails, gatewayOptions)
	if err != nil {
		return "", err, status, signals
	}

	// yaml.Marshal sorts the maps for deterministic YAML output
	// however, lists are kept in the order they were added, so we need to sort them manually,
	// to avoid any unexpected changes in the YAML output.
	slices.Sort(cfg.Service.Extensions)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err, status, signals
	}
	return string(data), nil, status, signals
}

//nolint:funlen,gocyclo // This function handles complex gateway configuration logic that is difficult to break down further
func CalculateGatewayConfig(
	dests []config.ExporterConfigurer,
	processors []config.ProcessorConfigurer,
	applySelfTelemetry func(c *config.Config, destinationPipelineNames []string, signalsRootPipelines []string) error,
	dataStreamsDetails []DataStreams,
	gatewayOptions *GatewayConfigOptions,
) (*config.Config, error, *config.ResourceStatuses, []common.ObservabilitySignal) {
	currentConfig := GetBasicConfig()

	configers, err := config.LoadConfigers()
	if err != nil {
		return nil, err, nil, nil
	}

	status := &config.ResourceStatuses{
		Destination: make(map[string]error),
		Processor:   make(map[string]error),
	}

	if _, exists := currentConfig.Receivers["otlp"]; !exists {
		return nil, fmt.Errorf("missing required receiver 'otlp' on config"), status, nil
	}

	// map of destination ID to list of forward connectors
	// this is used to build the data stream pipelines
	// e.g. { "destination-1": ["forward/traces/destination-1", "forward/metrics/destination-1", "forward/logs/destination-1"] }
	destForwardConnectors := make(map[string][]string)

	tracesEnabled := false
	metricsEnabled := false
	logsEnabled := false
	profilesEnabled := false

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

	if traceAggregationNeeded(gatewayOptions) {
		ensureGroupByTraceProcessor(currentConfig, &processorsResults, gatewayOptions)
	}

	// If tail sampling v2 is enabled, add the tail sampling processor to the traces processors.
	if gatewayOptions.TailSamplingEnabled != nil && *gatewayOptions.TailSamplingEnabled && gatewayOptions.OdigosConfigExtensionName != nil {
		processorsNames, processorsConfig := getTailSamplingProcessors(gatewayOptions)
		for name, cfg := range processorsConfig {
			currentConfig.Processors[name] = cfg
		}
		// apend processors to the front of the pipeline. this should be revisited.
		processorsResults.TracesProcessors = append(processorsNames, processorsResults.TracesProcessors...)
	}

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
				tracesEnabled = true
			case strings.HasPrefix(pipelineName, "metrics/"):
				metricsEnabled = true
			case strings.HasPrefix(pipelineName, "logs/"):
				logsEnabled = true
			case strings.HasPrefix(pipelineName, "profiles/"):
				profilesEnabled = true
			}

			// save the updated pipeline with the new receiver
			currentConfig.Service.Pipelines[pipelineName] = pipeline
		}

		status.Destination[dest.GetID()] = nil // mark this destination as success
	}
	// Profile destinations (e.g. Pyroscope) register their pipelines directly under
	// "profiles/<id>" and intentionally return no destination pipeline names from
	// ModifyConfig — so they bypass the forward-connector loop above. Detect them
	// by scanning registered pipelines so PROFILES is reflected in enabledSignals.
	if !profilesEnabled {
		for pipelineName := range currentConfig.Service.Pipelines {
			if strings.HasPrefix(pipelineName, "profiles/") {
				profilesEnabled = true
				break
			}
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
	if profilesEnabled {
		enabledSignals = append(enabledSignals, common.ProfilesObservabilitySignal)
	}

	if tracesEnabled {
		currentConfig.Processors[consts.OdigosTraceStateProcessorName] = config.GenericMap{}
	}

	//  Add pipelines that receive from routing connectors and forward to destinations
	dataStreamPipelines := buildDataStreamPipelines(dataStreamsDetails, destForwardConnectors)
	for name, pipe := range dataStreamPipelines {
		currentConfig.Service.Pipelines[name] = pipe
	}

	// Create root pipelines for each signal and connectors
	tracesPostForwardProcessors := processorsResults.TracesProcessorsPostSpanMetrics
	if tracesEnabled {
		tracesPostForwardProcessors = append(tracesPostForwardProcessors, consts.OdigosTraceStateProcessorName)
	}
	insertRootPipelinesToConfig(currentConfig,
		processorsResults.TracesProcessors,
		tracesPostForwardProcessors,
		processorsResults.MetricsProcessors,
		processorsResults.LogsProcessors,
		enabledSignals,
		gatewayOptions)

	// Optional: Add collector self-observability
	if applySelfTelemetry != nil {
		if err := applySelfTelemetry(currentConfig, unifiedDestinationPipelineNames, GetSignalsRootPipelineNames()); err != nil {
			return nil, err, status, nil
		}
	}

	// Defensive nil-checks to avoid panic on optional *bool fields.
	// Defaults:
	// - ServiceGraph.Disabled: assume false (enabled) if nil
	// - ClusterMetricsEnabled: assume false (disabled) if nil
	if tracesEnabled && (gatewayOptions.ServiceGraph.Disabled == nil || !*gatewayOptions.ServiceGraph.Disabled) {
		insertServiceGraphPipeline(currentConfig, gatewayOptions.ServiceGraph.ExtraDimensions, gatewayOptions.ServiceGraph.VirtualNodePeerAttributes)
	}
	if tracesEnabled {
		insertTraceCorrelationsServiceIOPipeline(currentConfig, gatewayOptions)
	}
	if metricsEnabled && gatewayOptions.ClusterMetricsEnabled != nil && *gatewayOptions.ClusterMetricsEnabled {
		insertClusterMetricsResources(currentConfig, gatewayOptions.OdigosNamespace)
	}

	// add the odigos config extension to the config if it is set
	// each platform (k8s, vm) will have a different extension name
	if gatewayOptions.OdigosConfigExtensionName != nil {
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, *gatewayOptions.OdigosConfigExtensionName)
		currentConfig.Extensions[*gatewayOptions.OdigosConfigExtensionName] = config.GenericMap{}
	}

	return currentConfig, nil, status, enabledSignals
}

func insertRootPipelinesToConfig(currentConfig *config.Config,
	tracesProcessors, tracesPostForwardProcessors, metricsProcessors, logsProcessors []string,
	signals []common.ObservabilitySignal, gatewayOptions *GatewayConfigOptions) {
	if slices.Contains(signals, common.TracesObservabilitySignal) {
		if traceAggregationNeeded(gatewayOptions) {
			applySplitTracesRootPipelines(currentConfig, tracesProcessors, tracesPostForwardProcessors, gatewayOptions.OdigosConfigExtensionName)
		} else {
			allTracesProcessors := append(slices.Clone(tracesProcessors), tracesPostForwardProcessors...)
			applyRootPipelineForSignal(currentConfig, common.TracesObservabilitySignal, allTracesProcessors, gatewayOptions.OdigosConfigExtensionName)
		}
	}

	if slices.Contains(signals, common.MetricsObservabilitySignal) {
		applyRootPipelineForSignal(currentConfig, common.MetricsObservabilitySignal, metricsProcessors, gatewayOptions.OdigosConfigExtensionName)
	}

	if slices.Contains(signals, common.LogsObservabilitySignal) {
		applyRootPipelineForSignal(currentConfig, common.LogsObservabilitySignal, logsProcessors, gatewayOptions.OdigosConfigExtensionName)
	}
}

// applySplitTracesRootPipelines forks traces after enrichment processors so complete trace batches
// are templated before tail sampling. traces/in aggregates and forwards; traces/exporting tail-samples,
// batches, and routes to destinations.
func applySplitTracesRootPipelines(
	currentConfig *config.Config,
	tracesProcessors, tracesPostForwardProcessors []string,
	odigosConfigExtensionName *string,
) {
	forwardConnectorName := consts.TracesPostGroupByForwardConnectorName
	currentConfig.Connectors[forwardConnectorName] = config.GenericMap{}

	tracesInProcessors, tracesExportingProcessors := splitTracesProcessorsForPipelines(tracesProcessors, tracesPostForwardProcessors)

	rootPipelineName := GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	currentConfig.Service.Pipelines[rootPipelineName] = config.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: append([]string{"resource/odigos-version"}, tracesInProcessors...),
		Exporters:  []string{forwardConnectorName},
	}

	routerConnectorName := fmt.Sprintf("odigosrouterconnector/%s", strings.ToLower(string(common.TracesObservabilitySignal)))
	routerConnectorCfg := config.GenericMap{}
	if odigosConfigExtensionName != nil {
		routerConnectorCfg["odigos_config_extension"] = *odigosConfigExtensionName
	}
	currentConfig.Connectors[routerConnectorName] = routerConnectorCfg

	currentConfig.Service.Pipelines[consts.TracesExportingPipelineName] = config.Pipeline{
		Receivers:  []string{forwardConnectorName},
		Processors: tracesExportingProcessors,
		Exporters:  []string{routerConnectorName},
	}
}

func splitTracesProcessorsForPipelines(tracesProcessors, tracesPostForwardProcessors []string) (tracesIn, tracesExporting []string) {
	exportingPipelineProcessors := map[string]struct{}{
		consts.OdigosTailSamplingProcessorName: {},
		consts.GenericBatchProcessorConfigKey:  {},
	}

	for _, processor := range tracesProcessors {
		if processor == consts.GroupByTraceProcessor {
			tracesIn = append(tracesIn, processor)
			continue
		}
		if _, isExportingProcessor := exportingPipelineProcessors[processor]; isExportingProcessor {
			tracesExporting = append(tracesExporting, processor)
			continue
		}
		tracesIn = append(tracesIn, processor)
	}

	tracesExporting = orderExportingPipelineProcessors(tracesExporting)
	return tracesIn, append(tracesExporting, tracesPostForwardProcessors...)
}

func orderExportingPipelineProcessors(processors []string) []string {
	ordered := make([]string, 0, len(processors))
	for _, processor := range []string{
		consts.OdigosTailSamplingProcessorName,
		consts.GenericBatchProcessorConfigKey,
	} {
		if slices.Contains(processors, processor) {
			ordered = append(ordered, processor)
		}
	}
	for _, processor := range processors {
		if processor == consts.OdigosTailSamplingProcessorName || processor == consts.GenericBatchProcessorConfigKey {
			continue
		}
		ordered = append(ordered, processor)
	}
	return ordered
}

func tracesPipelineForDownstreamConnectors(currentConfig *config.Config) string {
	if _, exists := currentConfig.Service.Pipelines[consts.TracesExportingPipelineName]; exists {
		return consts.TracesExportingPipelineName
	}
	return GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
}

func applyRootPipelineForSignal(currentConfig *config.Config, signal common.ObservabilitySignal,
	processors []string, odigosConfigExtensionName *string) {
	rootPipelineName := GetTelemetryRootPipelineName(signal)
	fullProcessors := append([]string{"resource/odigos-version"}, processors...)

	connectorName := fmt.Sprintf("odigosrouterconnector/%s", strings.ToLower(string(signal)))

	connectorCfg := config.GenericMap{}
	if odigosConfigExtensionName != nil {
		connectorCfg["odigos_config_extension"] = *odigosConfigExtensionName
	}
	currentConfig.Connectors[connectorName] = connectorCfg

	currentConfig.Service.Pipelines[rootPipelineName] = config.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: fullProcessors,
		Exporters:  []string{connectorName},
	}
}

func insertServiceGraphPipeline(currentConfig *config.Config, extraDimensions []string, virtualNodePeerAttributes []string) {
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

	// Build dimensions: always include service.name as the base, then append any extras
	dimensions := []string{string(semconv.ServiceNameKey)}
	dimensions = append(dimensions, extraDimensions...)

	// Add the service graph connector to receive the service graph metrics from the root traces pipeline
	// Retain incomplete edges for up to 15s to allow delayed span matching
	// Clean up every 5s to reduce memory pressure and avoid stale edges
	connectorCfg := config.GenericMap{
		"store": config.GenericMap{
			"ttl": "15s",
		},
		"store_expiration_loop": "5s",
		"dimensions":            dimensions,
	}

	// Only override virtual_node_peer_attributes when explicitly configured;
	// otherwise the connector uses its built-in defaults [peer.service, db.name, db.system].
	if len(virtualNodePeerAttributes) > 0 {
		connectorCfg["virtual_node_peer_attributes"] = virtualNodePeerAttributes
	}

	currentConfig.Connectors[consts.ServiceGraphConnectorName] = connectorCfg

	// Add the service graph pipeline to receive the service graph metrics from the root traces pipeline
	currentConfig.Service.Pipelines["metrics/servicegraph"] = config.Pipeline{
		Receivers: []string{consts.ServiceGraphConnectorName},
		Exporters: []string{"prometheus/servicegraph"},
	}

	// Add the service graph exporter to the traces pipeline that routes to destinations
	// (traces/exporting when the pipeline is split, otherwise traces/in).
	tracesPipelineName := tracesPipelineForDownstreamConnectors(currentConfig)
	pipeline, exists := currentConfig.Service.Pipelines[tracesPipelineName]
	if !exists {
		return
	}
	pipeline.Exporters = append(pipeline.Exporters, consts.ServiceGraphConnectorName)
	currentConfig.Service.Pipelines[tracesPipelineName] = pipeline
}

func insertTraceCorrelationsServiceIOPipeline(currentConfig *config.Config, gatewayOptions *GatewayConfigOptions) {
	cfg := gatewayOptions.TraceCorrelationsServiceIO
	if !common.TraceCorrelationsServiceIOPipelineActive(&common.TraceCorrelationsConfiguration{
		ServiceIO: cfg,
	}) {
		return
	}

	connectorCfg := config.GenericMap{}
	if len(cfg.InputSpanAttributes) > 0 {
		connectorCfg["input_span_attributes"] = cfg.InputSpanAttributes
	}
	if len(cfg.OutputSpanAttributes) > 0 {
		connectorCfg["output_span_attributes"] = cfg.OutputSpanAttributes
	}
	if cfg.MetricsFlushInterval != "" {
		connectorCfg["metrics_flush_interval"] = cfg.MetricsFlushInterval
	}
	if gatewayOptions.OdigosConfigExtensionName != nil {
		connectorCfg["odigos_config_extension"] = *gatewayOptions.OdigosConfigExtensionName
	}

	currentConfig.Connectors[consts.ServiceIOConnectorName] = connectorCfg

	exporterName := consts.TraceCorrelationsVictoriaMetricsExporterName
	currentConfig.Exporters[exporterName] = traceCorrelationsVictoriaMetricsExporter(gatewayOptions.OdigosNamespace)

	currentConfig.Service.Pipelines[consts.TraceCorrelationsMetricsPipelineName] = config.Pipeline{
		Receivers: []string{consts.ServiceIOConnectorName},
		Exporters: []string{exporterName},
	}

	tracesInPipelineName := GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	pipeline, exists := currentConfig.Service.Pipelines[tracesInPipelineName]
	if !exists {
		return
	}
	if !slices.Contains(pipeline.Exporters, consts.ServiceIOConnectorName) {
		pipeline.Exporters = append(pipeline.Exporters, consts.ServiceIOConnectorName)
	}
	currentConfig.Service.Pipelines[tracesInPipelineName] = pipeline
}

func traceCorrelationsVictoriaMetricsExporter(odigosNamespace string) config.GenericMap {
	endpoint := fmt.Sprintf(
		"http://%s.%s:8428/opentelemetry",
		consts.TraceCorrelationsMetricsServiceName,
		odigosNamespace,
	)
	return config.GenericMap{
		"endpoint": endpoint,
		"retry_on_failure": config.GenericMap{
			"enabled": false,
		},
		"tls": config.GenericMap{
			"insecure": true,
		},
	}
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
		},
		Exporters: map[string]interface{}{},
		Service: config.Service{
			Pipelines:  map[string]config.Pipeline{},
			Extensions: []string{"health_check", "pprof"},
		},
	}
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

const defaultTraceAggregationWaitDuration = "30s"

func traceAggregationNeeded(gatewayOptions *GatewayConfigOptions) bool {
	if gatewayOptions.TailSamplingEnabled != nil && *gatewayOptions.TailSamplingEnabled {
		return true
	}

	if common.TraceCorrelationsServiceIOPipelineActive(&common.TraceCorrelationsConfiguration{
		ServiceIO: gatewayOptions.TraceCorrelationsServiceIO,
	}) {
		return true
	}

	return false
}

func ensureGroupByTraceProcessor(
	currentConfig *config.Config,
	processorsResults *config.CrdProcessorResults,
	gatewayOptions *GatewayConfigOptions,
) {
	if slices.Contains(processorsResults.TracesProcessors, consts.GroupByTraceProcessor) {
		return
	}

	waitDuration := defaultTraceAggregationWaitDuration
	if gatewayOptions.TraceAggregationWaitDuration != nil && *gatewayOptions.TraceAggregationWaitDuration != "" {
		waitDuration = *gatewayOptions.TraceAggregationWaitDuration
	}

	currentConfig.Processors[consts.GroupByTraceProcessor] = config.GenericMap{
		"wait_duration": waitDuration,
	}
	processorsResults.TracesProcessors = append(
		[]string{consts.GroupByTraceProcessor},
		processorsResults.TracesProcessors...,
	)
}

func getTailSamplingProcessors(gatewayOptions *GatewayConfigOptions) ([]string, map[string]config.GenericMap) {
	tailSamplingProcessorCfg := config.GenericMap{
		"odigos_config_extension": *gatewayOptions.OdigosConfigExtensionName,
	}
	if gatewayOptions.SamplingDryRun {
		tailSamplingProcessorCfg["dry_run"] = true
	}
	if gatewayOptions.SamplingSpanAttributes != nil {
		spanSamplingAttributesCfg := config.GenericMap{}
		if gatewayOptions.SamplingSpanAttributes.Disabled != nil {
			spanSamplingAttributesCfg["disabled"] = *gatewayOptions.SamplingSpanAttributes.Disabled
		}
		if gatewayOptions.SamplingSpanAttributes.SamplingCategoryDisabled != nil {
			spanSamplingAttributesCfg["sampling_category_disabled"] = *gatewayOptions.SamplingSpanAttributes.SamplingCategoryDisabled
		}
		if gatewayOptions.SamplingSpanAttributes.TraceDecidingRuleDisabled != nil {
			spanSamplingAttributesCfg["trace_deciding_rule_disabled"] = *gatewayOptions.SamplingSpanAttributes.TraceDecidingRuleDisabled
		}
		if gatewayOptions.SamplingSpanAttributes.SpanDecisionAttributesDisabled != nil {
			spanSamplingAttributesCfg["span_decision_attributes_disabled"] = *gatewayOptions.SamplingSpanAttributes.SpanDecisionAttributesDisabled
		}
		tailSamplingProcessorCfg["span_sampling_attributes"] = spanSamplingAttributesCfg
	}

	processors := map[string]config.GenericMap{
		consts.OdigosTailSamplingProcessorName: tailSamplingProcessorCfg,
	}

	return []string{consts.OdigosTailSamplingProcessorName}, processors
}
