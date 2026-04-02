package common

// OpenTelemetry component instance names for continuous profiling pipelines. Keys must be unique
// within each collector's merged config (node collector and cluster gateway are separate binaries).
const (
	// ProfilingReceiver is the contrib profiling receiver on the node collector.
	ProfilingReceiver = "profiling"

	// Node collector profiles domain — receive on host, forward to cluster gateway.
	ProfilingNodeFilterProcessor        = "filter/profiles-node"
	ProfilingNodeK8sAttributesProcessor = "k8s_attributes/profiles-node"
	ProfilingNodeToGatewayExporter      = "otlp_grpc/profiles-to-gateway"

	// Cluster gateway profiles pipeline — OTLP in from nodes, export to UI (no extra processors).
	ProfilingGatewayToUIExporter = "otlp_grpc/profiles-to-ui"
)
