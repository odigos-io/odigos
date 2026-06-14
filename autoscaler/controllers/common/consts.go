package common

// OpenTelemetry component instance names for continuous profiling pipelines. Keys must be unique
// within each collector's merged config (node collector and cluster gateway are separate binaries).
const (
	// ProfilingReceiver is the contrib profiling receiver on the node collector.
	ProfilingReceiver = "profiling"

	// Node collector profiles domain — receive on host, forward to cluster gateway.
	ProfilingNodeFilterProcessor        = "filter/profiles-node"
	ProfilingNodeK8sAttributesProcessor = "k8s_attributes/profiles-node"
	// ProfilingNodeOdigosProfilesProcessor keeps only profiles for workloads present in odigos_config_k8s (InstrumentationConfig).
	ProfilingNodeOdigosProfilesProcessor = "odigosprofilesprocessor/profiles-node"
	// ProfilingNodeServiceNameProcessor sets service.name from K8s metadata so Pyroscope
	// (and other backends) show workload names instead of unknown_service:<process>.
	ProfilingNodeServiceNameProcessor = "transform/profiles-service-name"
	ProfilingNodeToGatewayExporter    = "otlp_grpc/profiles-to-gateway"

	// Cluster gateway profiles pipeline — OTLP in from nodes, export to UI (no extra processors).
	ProfilingGatewayToUIExporter = "otlp_grpc/profiles-to-ui"
)

// OpenTelemetry component instance name for the side-channel exporter
// appended to the root traces pipeline on the cluster gateway. Identifier
// uses a neutral "Insights" stem; the string VALUE retains its established
// form so the rendered collector configmap stays backwards-compatible.
const (
	// InsightsGatewayExporter forwards spans to the in-cluster sidecar service.
	InsightsGatewayExporter = "otlp_grpc/security"
)
