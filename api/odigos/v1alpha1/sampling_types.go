package v1alpha1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// define conditions to match specific sources (containers) managed by odigos.
// a source container matches, if ALL non empty fields match (AND semantics)
//
// common patterns:
// - Specific kubernetes workload by name (WorkloadNamespace + WorkloadKind + WorkloadName) - all containers (usually there is only one with agent injection)
// - Specific container in a kubernetes workload (WorkloadNamespace + WorkloadKind + WorkloadName + ContainerName) - only this container
// - All services in a kubernetes namespace (WorkloadNamespace) - all containers in all sources in the namespace
// - All services implemented in a specific programming language (WorkloadLanguage) - all container which are running odigos agent for this language
type SourcesScope struct {
	WorkloadName      string                 `json:"workloadName,omitempty"`
	WorkloadKind      k8sconsts.WorkloadKind `json:"workloadKind,omitempty"`
	WorkloadNamespace string                 `json:"workloadNamespace,omitempty"`
	ContainerName     string                 `json:"containerName,omitempty"`

	WorkloadLanguage common.ProgrammingLanguage `json:"workloadLanguage,omitempty"`
}

// match operations for tail sampling with the full context of the span.
// this is used by sampling rules to limit it only to specific operations.
// if the rule matches a sapn, the behavior is determined by the rule itself.
type TailSamplingOperationMatcher struct {

	// match http server operations in a generic way.
	HttpServer *HttpServerOperationMatcher `json:"httpServer,omitempty"`

	// match kafka consumer operations (consume spans)
	KafkaConsumer *KafkaOperationMatcher `json:"kafkaConsumer,omitempty"`

	// match kafka producer operations (produce spans)
	KafkaProducer *KafkaOperationMatcher `json:"kafkaProducer,omitempty"`
}

// can match a specific operation for head sampling.
// head sampling has access only to the attributes available at span start time,
// and sampling decisions can only be based on the root span of the trace.
type HeadSamplingOperationMatcher struct {
	// match http server operation (trace that starts with an http endpoint)
	HttpServer *HeadSamplingHttpServerOperationMatcher `json:"httpServer,omitempty"`

	// match http client operation (trace that starts with an http client operation)
	// common for agents that are internally calling home over http, and exporting data.
	HttpClient *HeadSamplingHttpClientOperationMatcher `json:"httpClient,omitempty"`
}

// match http server operations for noisy operations matching (only attributes available at span start time)
type HeadSamplingHttpServerOperationMatcher struct {
	// match route exactly
	Route string `json:"route,omitempty"`
	// match preffix of route
	RoutePrefix string `json:"routePrefix,omitempty"`
	// match method exactly, can be empty to match any method
	Method string `json:"method,omitempty"`
}

// match http client operations for noisy operations matching (only attributes available at span start time)
// can be used to filter out outgoing http requests for other agents calling home or exporting data.
type HeadSamplingHttpClientOperationMatcher struct {
	// match server address exactly (e.g. collector.my.vendor.com)
	ServerAddress string `json:"serverAddress,omitempty"`
	// match url path exactly (e.g. /api/v1/metrics)
	UrlPath string `json:"urlPath,omitempty"`
	// match method exactly, can be empty to match any method
	Method string `json:"method,omitempty"`
}

// endpoints which are considered "noise", and provide no or very little observability value.
// these traces should not be collected at all, or dropped aggresevly.
// motivation is data sentization and performance improvment (even if cost is not a factor)
//
// examples:
// - health-checks (readiness and liveness probes)
// - metrics scrape endpoints (promethues /metrics endpoint)
// - other agents calling home (outgoing http requests to collector.my.vendor.com)
type NoisyOperations struct {
	// limit this rule to specific sources (by name, namespace, language, etc.)
	// for example: if "other agent" rule for noisty operation is relevant only in java,
	// limit this rule by setting source scope to java and prevent other languages from being affected
	// if the list is empty - all sources are matched.
	SourceScopes []SourcesScope `json:"sourceScopes,omitempty"`

	// limit this rule to specific operations.
	// for example: specific http server endpoint (GET "/healthz" as an example).
	// this field is optional, and if not set, the rule will be applied to all operations.
	Operation *HeadSamplingOperationMatcher `json:"operation,omitempty"`

	// sampling percentage for noisy operations.
	// if unset, 0% of such the traces will be collected.
	// this percentage has "at most" semantics - the final sampling percentage for traces that match this rule
	// will be the highest possible, but at most this value.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	PercentageAtMost *float64 `json:"percentageAtMost,omitempty"`

	// optional free-form text field that allows you to attach notes
	// for future context and maintenance.
	// users can write why this rule was added, observations, document considerations, etc.
	Notes string `json:"notes,omitempty"`
}

// match only http server spans for a specific endpoint.
// user can specify route and method to match, and limit a sampling instruction to only this operation.
type HttpServerOperationMatcher struct {

	// a specific exact match http route
	Route string `json:"route,omitempty"`

	// any route that starts with a specific prefix
	RoutePrefix string `json:"routePrefix,omitempty"`

	// optionally limit to specific http method
	Method string `json:"method,omitempty"`
}

// match a kafka consumer or producer operation for a specific topic.
type KafkaOperationMatcher struct {

	// the topic name to match.
	// if left empty, all topics are matched.
	KafkaTopic string `json:"kafkaTopic,omitempty"`
}

// define operations (spans) with high observability value.
// if found anywhere in the trace, the entire trace will be kept
// regaradless of any cost reduction rules.
type HighlyRelevantOperation struct {

	// limit the operation to specific sources.
	// an empty list will match any source.
	// if multiple items are set, the operation match if any one matches
	// this relates to the "ResourceAttributes" part of a span.
	SourceScopes []SourcesScope `json:"sourceScopes,omitempty"`

	// if "Error" is set to true, only spans with SpanStatus set to "Error" are considered
	Error bool `json:"error,omitempty"`

	// if Duration is set, only operations with duration in milli seconds larger then this value are considered
	DurationAtLeastMs *int `json:"durationAtLeastMs,omitempty"`

	// optionally, limit this rule to specific operations.
	// for example: specific endpoint or kafka topic.
	// this field is optional, and if not set, the rule will be applied to all operations.
	Operation *TailSamplingOperationMatcher `json:"operation,omitempty"`

	// traces that contains this operation will be sampled by at least this percentage.
	// if unset, 100% of such the traces will be sampled.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	PercentageAtLeast *float64 `json:"percentageAtLeast,omitempty"`

	// optional free-form text field that allows you to attach notes
	// for future context and maintenance.
	// users can write why this rule was added, observations, document considerations, etc.
	Notes string `json:"notes,omitempty"`
}

type CostReductionRule struct {

	// limit this rule to specific sources (by name, namespace, language, etc.)
	// an empty list will match any source.
	// if multiple items are set, the operation match if any one matches
	// this relates to the "ResourceAttributes" part of a span.
	SourceScopes []SourcesScope `json:"sourceScopes,omitempty"`

	// limit this rule to specific operations.
	// for example: specific endpoint or kafka topic.
	// this field is optional, and if not set, the rule will be applied to all operations.
	Operation *TailSamplingOperationMatcher `json:"operation,omitempty"`

	// sampling percentage for cost reduction.
	// this field is required.
	// the final sampling percentage for traces that match this rule
	// will be the highest possible, but at most this value.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Required
	PercentageAtMost float64 `json:"percentageAtMost"`

	// optional free-form text field that allows you to attach notes
	// for future context and maintenance.
	// users can write why this rule was added, observations, document considerations, etc.
	Notes string `json:"notes,omitempty"`
}

// define sampling rules.
// the rules can be defined as one or multiple objects in kubernetes,
// and are all joined together to form the global sampling rules.
// odigos users can group rules based on whatever criteria that makes sense for them,
// for example - by team, by client, by usecase, admin-policy, etc.
type SamplingSpec struct {

	// give these sampling rules a name for display, easier identification and reference.
	Name string `json:"name,omitempty"`

	// a free-form text field that allows you to attach notes regardinag the rule for convenience.
	// Odigos does not use or assume any meaning from this field.
	Notes string `json:"notes,omitempty"`

	// if set to true, the sampling rules will be disabled,
	// they will not be taken into account for any sampling decisions.
	// useful if you want to temporarily disable the rules but re-enable them later,
	Disabled                 bool                      `json:"disabled,omitempty"`
	NoisyOperations          []NoisyOperations         `json:"noisyOperations,omitempty"`
	HighlyRelevantOperations []HighlyRelevantOperation `json:"highlyRelevantOperations,omitempty"`
	CostReductionRules       []CostReductionRule       `json:"costReductionRules,omitempty"`
}

// SamplingStatus defines the observed state of Sampling.
type SamplingStatus struct {
	// Represents the observations of a Sampling's current state.
	// Known .status.conditions.type are: "Available", "Progressing"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// Sampling is the Schema for the sampling rules API.
type Sampling struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SamplingSpec   `json:"spec,omitempty"`
	Status SamplingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SamplingList contains a list of Sampling.
type SamplingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Sampling `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Sampling{}, &SamplingList{})
}

func ComputeNoisyOperationHash(rule *NoisyOperations) string {
	ruleFields := NoisyOperations{
		SourceScopes: rule.SourceScopes,
		Operation:    rule.Operation,
		// PercentageAtMost can be changed without affecting the rule id
		// notes are not effecting the rule id
	}
	uniqueRuleBytes, _ := json.Marshal(ruleFields)
	h := sha256.New()
	h.Write(uniqueRuleBytes)
	return hex.EncodeToString(h.Sum(nil)[:8])
}

// compute unique id for the rule - which can be used to reference.
func ComputeHighlyRelevantOperationHash(rule *HighlyRelevantOperation) string {

	// copy just those fields that are relevant for the rule id
	ruleFields := HighlyRelevantOperation{
		SourceScopes:      rule.SourceScopes,
		Error:             rule.Error,
		DurationAtLeastMs: rule.DurationAtLeastMs,
		Operation:         rule.Operation,
		// PercentageAtLeast can be changed without affecting the rule id
		// notes are not effecting the rule id
	}

	uniqueRuleBytes, _ := json.Marshal(ruleFields)
	h := sha256.New()
	h.Write(uniqueRuleBytes)
	return hex.EncodeToString(h.Sum(nil)[:8])
}

// compute unique id for the rule - which can be used to reference.
func ComputeCostReductionRuleHash(rule *CostReductionRule) string {
	ruleFields := CostReductionRule{
		SourceScopes: rule.SourceScopes,
		Operation:    rule.Operation,
		// PercentageAtMost can be changed without affecting the rule id
		// notes are not effecting the rule id
	}
	uniqueRuleBytes, _ := json.Marshal(ruleFields)
	h := sha256.New()
	h.Write(uniqueRuleBytes)
	return hex.EncodeToString(h.Sum(nil)[:8])
}
