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
	ServiceName string `json:"serviceName,omitempty"`

	WorkloadName      string `json:"workloadName,omitempty"`
	WorkloadKind      string `json:"workloadKind,omitempty"`
	WorkloadNamespace string `json:"workloadNamespace,omitempty"`
	ContainerName     string `json:"containerName,omitempty"`

	WorkloadLanguage string `json:"workloadLanguage,omitempty"`
}

type OperationMatcher struct {
	HttpServer *HttpServerOperationMatcher `json:"httpServer,omitempty"`
	Kafka      *KafkaOperationMatcher      `json:"kafka,omitempty"`
}

// endpoints which are considered "noise", and provide no or very little observability value.
// these traces should not be collected at all, or dropped aggresevly.
// motivation is data sentization and performance improvment (even if cost is not a factor)
//
// examples:
// - health-checks (readiness and liveness probes)
// - metrics scrape endpoints (promethues /metrics endpoint)
type NoisyEndpoint struct {
	Services         []Services `json:"services,omitempty"`
	HttpRoute        string     `json:"httpRoute,omitempty"`
	HttpMethod       string     `json:"httpMethod,omitempty"`
	PercentageAtMost *float64   `json:"percentageAtMost,omitempty"`
	Notes            string     `json:"notes,omitempty"`
}

type HttpServerOperationMatcher struct {

	// a specific exact match http route
	Route string `json:"route,omitempty"`

	// any route that starts with a specific prefix
	RoutePrefix string `json:"routePrefix,omitempty"`

	// optionally limit to specific http method
	Method string `json:"method,omitempty"`
}

type KafkaOperationMatcher struct {
	KafkaTopic string `json:"kafkaTopic,omitempty"`
}

// define operations (spans) with high observability value.
// if found anywhere in the trace, the entire trace will be kept
// regaradless of any cost reduction rules.
type HighlyRelevantOperation struct {

	// limit the operation to specific services.
	// an empty list will match any service.
	// if multiple items are set, the operation match if any one matches
	// this relates to the "ResourceAttributes" part of a span.
	Services []Services `json:"services,omitempty"`

	// if "Error" is set to true, only spans with SpanStatus set to "Error" are considered
	Error bool `json:"error,omitempty"`

	// if Duration is set, only operations with duration in milli seconds larger then this value are considered
	DurationMsAtLeast *int `json:"durationMsAtLeast,omitempty"`

	Operation *OperationMatcher `json:"operation,omitempty"`

	// traces that contains this operation will be sampled by at least this percentage.
	// if unset, 100% of such the traces will be sampled.
	PercentageAtLeast *float64 `json:"percentageAtLeast,omitempty"`

	Notes string `json:"notes,omitempty"`
}

type CostReductionRule struct {
	Services         []Services       `json:"services,omitempty"`
	Operation        OperationMatcher `json:"operation,omitempty"`
	PercentageAtMost *float64         `json:"percentageAtMost,omitempty"`
	Notes            string           `json:"notes,omitempty"`
}

// define sampling rules.
// the rules can be defined as one or multiple objects in kubernetes,
// and are all joined together to form the global sampling rules.
// odigos users can group rules based on whatever criteria that makes sense for them,
// for example - by team, by client, by usecase, admin-policy, etc.
type SamplingSpec struct {

	// give these sampling rules a name for display, easier identification and reference.
	Name string `json:"name,omitempty"`

	// a free-form text field that allows you to attach notes regarding the rule for convenience.
	// Odigos does not use or assume any meaning from this field.
	Notes string `json:"notes,omitempty"`

	// if set to true, the sampling rules will be disabled,
	// they will not be taken into account for any sampling decisions.
	// useful if you want to temporarily disable the rules but re-enable them later,
	Disabled                 bool                      `json:"disabled,omitempty"`
	NoisyEndpoints           []NoisyEndpoint           `json:"noisyEndpoints,omitempty"`
	HighlyRelevantOperations []HighlyRelevantOperation `json:"highlyRelevantOperations,omitempty"`
	CostReductionRules       []CostReductionRule       `json:"costReductionRules,omitempty"`
}
