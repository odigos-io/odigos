package v1alpha1

// define conditions to match specific services in the cluster.
// a service matches, if ALL non empty fields match (AND semantics)
//
// common patterns:
// - Specific service by name (service.name)
// - Specific kubernetes workload by name (WorkloadNamespace + WorkloadKind + WorkloadName)
// - Specific container in a kubernetes workload (WorkloadNamespace + WorkloadKind + WorkloadName + ContainerName)
// - All services in a kubernetes namespace (WorkloadNamespace)
// - All services implemented in a specific programming language
type Services struct {
	ServiceName string

	WorkloadName      string
	WorkloadKind      string
	WorkloadNamespace string
	ContainerName     string

	WorkloadLanguage string
}

type OperationMatcher struct {
	httpServer *HttpServerOperationMatcher
	kafka      *KafkaOperationMatcher
}

// endpoints which are considered "noise", and provide no or very little observability value.
// these traces should not be collected at all, or dropped aggresevly.
// motivation is data sentization and performance improvment (even if cost is not a factor)
//
// examples:
// - health-checks (readiness and liveness probes)
// - metrics scrape endpoints (promethues /metrics endpoint)
type NoisyEndpoint struct {
	Services         []Services
	HttpRoute        string
	HttpMethod       string
	PercentageAtMost *float64
	Notes            string
}

type HttpServerOperationMatcher struct {

	// a specific exact match http route
	Route string

	// any route that starts with a specific prefix
	RoutePrefix string

	// optionally limit to specific http method
	Method string
}

type KafkaOperationMatcher struct {
	KafkaTopic string
}

// define operations (spans) with high observability value.
// if found anywhere in the trace, the entire trace will be kept
// regaradless of any cost reduction rules.
type HighlyRelevantOperation struct {

	// limit the operation to specific services.
	// an empty list will match any service.
	// if multiple items are set, the operation match if any one matches
	// this relates to the "ResourceAttributes" part of a span.
	Services []Services

	// if "Error" is set to true, only spans with SpanStatus set to "Error" are considered
	Error bool

	// if Duration is set, only operations with duration in milli seconds larger then this value are considered
	DurationMsAtLeast *int

	Operation *OperationMatcher

	// traces that contains this operation will be sampled by at least this percentage.
	// if unset, 100% of such the traces will be sampled.
	PercentageAtLeast *float64

	Notes string
}

type CostReductionRule struct {
	Services         []Services
	Operation        OperationMatcher
	PercentageAtMost float64
	Notes            string
}

type SamplingSpec struct {
	NoisyEndpoints           []NoisyEndpoint
	HighlyRelevantOperations []HighlyRelevantOperation
	CostReductionRules       []CostReductionRule
}

// type TailSamplingConfig struct {

// 	// if set, tail sampling will be disabled, regardless of any sampling rules that requires it.
// 	// tail sampling requires additional resources in the collector and introduces latency in trace processing.
// 	Disabled bool

// 	TraceAggregationWaitDuration *float64
// }

// type HeadSamplingConfig struct {

// 	// if set to true, all the kubelet health probes (startup, readiness and liveness probes)
// 	// will automatically be detected and added to the "noisy endpoints" to be dropped.
// 	AutoIgnoreHalthProbes bool
// }
