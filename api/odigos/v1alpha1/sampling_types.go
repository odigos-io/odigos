package v1alpha1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SourcesScope is defined in api/k8sconsts so that both api and k8sutils can use it
// without a circular module dependency (k8sutils imports api).
type SourcesScopes = k8sconsts.SourcesScopes

// endpoints (or other operations) which are considered "noise", and provide no or very little observability value.
// these traces should not be collected at all, or dropped aggresevly.
// motivation is data sentization and performance improvment (even if cost is not a factor)
//
// examples:
// - health-checks (readiness and liveness probes)
// - metrics scrape endpoints (prometheus /metrics endpoint)
// - other agents calling home (outgoing http requests to collector.my.vendor.com)
type NoisyOperation struct {
	// user provided name, for easier identification and reference.
	// use short and descriptive name, like "health check", "always keep /transaction in payment service", etc.
	// odigos does not use or assume any meaning from this field,
	// but it is written as metric attribute, and stored as span attribute on participating spans.
	Name string `json:"name,omitempty"`

	// if set to true, the rule will be disabled,
	// e.g. will not be taken into account for any sampling decisions.
	// disabled rules still participate in metrics calculations,
	// allowing enhanced tools and data for troubleshooting and sampling maintenance.
	Disabled bool `json:"disabled,omitempty"`

	// limit this rule to specific sources (by name, namespace, language, etc.)
	// for example: if "other agent" rule for noisty operation is relevant only in java,
	// limit this rule by setting source scope to java and prevent other languages from being affected
	// if the list is empty - all sources are matched.
	SourceScopes *SourcesScopes `json:"sourceScopes,omitempty"`

	// limit this rule to specific operations.
	// for example: specific http server endpoint (GET "/healthz" as an example).
	// this field is optional, and if not set, the rule will be applied to all operations.
	Operation *commonapisampling.HeadSamplingOperationMatcher `json:"operation,omitempty"`

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

	// raise this rule's percentage while a service its traces reach has a spike in a RED metric.
	// the rule's percentage is restored once the metric recovers.
	// each episode is recorded in the status of the Sampling, then in odigos insights once it ends.
	Surge *TraceSurgeSettings `json:"surge,omitempty"`
}

// TraceSurgeMetric is the RED metric of a service's server spans that a trace surge watches.
// +kubebuilder:validation:Enum=error_rate;latency_p95;request_rate
type TraceSurgeMetric string

const (
	// percent of server calls whose span has an error status.
	TraceSurgeMetricErrorRate TraceSurgeMetric = "error_rate"
	// 95th percentile of server call duration, in milliseconds.
	TraceSurgeMetricLatencyP95 TraceSurgeMetric = "latency_p95"
	// server calls per second.
	TraceSurgeMetricRequestRate TraceSurgeMetric = "request_rate"
)

// TraceSurgeSettings raise a noisy operation's percentage while a service has a spike.
// every service the rule's traces reach is evaluated on its own, from span metrics recorded before sampling.
// a spike in a service raises the percentage of the workloads in the rule's scope whose traces lead to it.
type TraceSurgeSettings struct {
	// changes on every edit of the settings, so that concurrent edits can be detected.
	Version string `json:"version,omitempty"`

	Metric TraceSurgeMetric `json:"metric"`

	// the metric must exceed this value to start a surge:
	// percent for error_rate, milliseconds for latency_p95, requests per second for request_rate.
	// +kubebuilder:validation:Minimum=0
	Threshold float64 `json:"threshold"`

	// how long the metric must stay above the threshold before the percentage is raised.
	// +kubebuilder:validation:Minimum=1
	SustainedSeconds int `json:"sustainedSeconds"`

	// a service with fewer server calls than this in the evaluation window is not evaluated.
	// +kubebuilder:validation:Minimum=1
	MinimumRequests int `json:"minimumRequests"`

	// the percentage of new traces to sample during a surge.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	BoostPercent float64 `json:"boostPercent"`

	// the surge ends once the metric stays at or below this value for recoverySeconds.
	// +kubebuilder:validation:Minimum=0
	RecoveryThreshold float64 `json:"recoveryThreshold"`

	// +kubebuilder:validation:Minimum=1
	RecoverySeconds int `json:"recoverySeconds"`

	// the percentage stays raised at least this long, however soon the metric recovers.
	// +kubebuilder:validation:Minimum=1
	MinimumBoostSeconds int `json:"minimumBoostSeconds"`
}

// define operations (spans) with high observability value.
// if found anywhere in the trace, the entire trace will be kept
// regaradless of any cost reduction rules.
type HighlyRelevantOperation struct {

	// user provided name, for easier identification and reference.
	// use short and descriptive name, like "health check", "always keep /transaction in payment service", etc.
	// odigos does not use or assume any meaning from this field,
	// but it is written as metric attribute, and stored as span attribute on participating spans.
	Name string `json:"name,omitempty"`

	// if set to true, the rule will be disabled,
	// e.g. will not be taken into account for any sampling decisions.
	// disabled rules still participate in metrics calculations,
	// allowing enhanced tools and data for troubleshooting and sampling maintenance.
	Disabled bool `json:"disabled,omitempty"`

	// limit the operation to specific sources.
	// an empty list will match any source.
	// if multiple items are set, the operation match if any one matches
	// this relates to the "ResourceAttributes" part of a span.
	SourceScopes *SourcesScopes `json:"sourceScopes,omitempty"`

	// if "Error" is set to true, only spans with SpanStatus set to "Error" are considered
	Error bool `json:"error,omitempty"`

	// if Duration is set, only operations with duration in milli seconds larger then this value are considered
	DurationAtLeastMs *int `json:"durationAtLeastMs,omitempty"`

	// optionally, limit this rule to specific operations.
	// for example: specific endpoint or kafka topic.
	// this field is optional, and if not set, the rule will be applied to all operations.
	Operation *commonapisampling.TailSamplingOperationMatcher `json:"operation,omitempty"`

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
	// user provided name, for easier identification and reference.
	// use short and descriptive name, like "health check", "always keep /transaction in payment service", etc.
	// odigos does not use or assume any meaning from this field,
	// but it is written as metric attribute, and stored as span attribute on participating spans.
	Name string `json:"name,omitempty"`

	// if set to true, the rule will be disabled,
	// e.g. will not be taken into account for any sampling decisions.
	// disabled rules still participate in metrics calculations,
	// allowing enhanced tools and data for troubleshooting and sampling maintenance.
	Disabled bool `json:"disabled,omitempty"`

	// limit this rule to specific sources (by name, namespace, language, etc.)
	// an empty list will match any source.
	// if multiple items are set, the operation match if any one matches
	// this relates to the "ResourceAttributes" part of a span.
	SourceScopes *SourcesScopes `json:"sourceScopes,omitempty"`

	// limit this rule to specific operations.
	// for example: specific endpoint or kafka topic.
	// this field is optional, and if not set, the rule will be applied to all operations.
	Operation *commonapisampling.TailSamplingOperationMatcher `json:"operation,omitempty"`

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

	NoisyOperations          []NoisyOperation          `json:"noisyOperations,omitempty"`
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

	// The trace surges of the rules that are open: limited, boosting or recovering. A surge that
	// ended stays until its targets are confirmed back, or, when it ended at the maximum duration,
	// until its service recovers. Odigos insights keeps the surges after that.
	// +optional
	TraceSurges []TraceSurge `json:"traceSurges,omitempty"`
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

func ComputeNoisyOperationHash(rule *NoisyOperation) string {
	ruleFields := NoisyOperation{
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
