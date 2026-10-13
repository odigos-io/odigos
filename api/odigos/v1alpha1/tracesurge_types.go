package v1alpha1

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=Limited;Boosting;Recovering;Restored
type TraceSurgePhase string

const (
	// the metric crossed the threshold, but raising sampling would exceed the trace surge limits
	// (sampling.traceSurge in the odigos configuration), so nothing was raised.
	// the surge starts boosting if the limits allow it while the metric is still above the threshold.
	TraceSurgePhaseLimited TraceSurgePhase = "Limited"
	// the metric is above the recovery threshold, and the targets sample at the boost percentage.
	TraceSurgePhaseBoosting TraceSurgePhase = "Boosting"
	// the metric is at or below the recovery threshold.
	// the targets sample at the boost percentage until the recovery window and the minimum boost duration have passed.
	TraceSurgePhaseRecovering TraceSurgePhase = "Recovering"
	// the targets are back at the rule's normal percentage.
	TraceSurgePhaseRestored TraceSurgePhase = "Restored"
)

// a workload whose sampling a trace surge raises.
type TraceSurgeTarget struct {
	Workload k8sconsts.PodWorkload `json:"workload"`

	// set when the target was chosen without evidence that its traces reach the service,
	// e.g. when the service graph has no edge for it yet.
	BroadScopeReason string `json:"broadScopeReason,omitempty"`
}

type TraceSurgeSpec struct {
	// the Sampling object that holds the rule.
	SamplingName string `json:"samplingName"`

	// the id of the noisy operation rule (ComputeNoisyOperationHash).
	RuleID string `json:"ruleId"`

	RuleName string `json:"ruleName,omitempty"`

	// the rule's surge settings when the surge started.
	Settings TraceSurgeSettings `json:"settings"`

	// the rule's percentage outside a surge.
	NormalPercent float64 `json:"normalPercent"`

	// the service whose metric crossed the threshold.
	Service k8sconsts.PodWorkload `json:"service"`

	// the workloads in the rule's scope whose traces lead to the service.
	Targets []TraceSurgeTarget `json:"targets,omitempty"`
}

// one evaluation of a service's metric.
type TraceSurgeObservation struct {
	At metav1.Time `json:"at"`

	// in the unit of the rule's metric.
	Value float64 `json:"value"`

	// server calls in the evaluation window, of the operation if set.
	Requests int64 `json:"requests"`

	// the length of the evaluation window the value was measured over.
	WindowSeconds int `json:"windowSeconds,omitempty"`

	// the server operation (span name, e.g. "GET /checkout/{id}") whose value this is, when one
	// operation's value is above the whole service's. Empty for all of the service's server spans.
	Operation string `json:"operation,omitempty"`
}

// what an instrumented process reports it applied.
type TraceSurgeInstanceStatus struct {
	// the InstrumentationInstance of the process.
	Name string `json:"name"`

	Pod string `json:"pod"`

	// +kubebuilder:validation:Enum=confirmed;unknown;failed
	State string `json:"state"`

	// the rule's percentage the process reports, if any.
	AppliedPercent *float64 `json:"appliedPercent,omitempty"`

	At *metav1.Time `json:"at,omitempty"`

	Message string `json:"message,omitempty"`
}

type TraceSurgeTargetStatus struct {
	Workload k8sconsts.PodWorkload `json:"workload"`

	// processes that report the percentage the surge requires.
	Confirmed int `json:"confirmed"`

	// instrumented processes of the workload.
	Total int `json:"total"`

	// when every process first reported the required percentage.
	ConfirmedAt *metav1.Time `json:"confirmedAt,omitempty"`

	Instances []TraceSurgeInstanceStatus `json:"instances,omitempty"`
}

type TraceSurgeEvent struct {
	At          metav1.Time `json:"at"`
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
}

type TraceSurgeStatus struct {
	Phase TraceSurgePhase `json:"phase,omitempty"`

	// the observation that started the surge.
	Trigger *TraceSurgeObservation `json:"trigger,omitempty"`

	// the observations of the sustained window that led to the surge.
	TriggerSamples []TraceSurgeObservation `json:"triggerSamples,omitempty"`

	// the latest evaluation of the service's metric.
	LastObservation *TraceSurgeObservation `json:"lastObservation,omitempty"`

	BoostedAt *metav1.Time `json:"boostedAt,omitempty"`

	// when the metric last went to or below the recovery threshold, while the surge is recovering.
	RecoveryStartedAt *metav1.Time `json:"recoveryStartedAt,omitempty"`

	RestoredAt *metav1.Time `json:"restoredAt,omitempty"`

	// why the normal percentage was restored, e.g. the metric recovered or the rule was removed.
	RestoreReason string `json:"restoreReason,omitempty"`

	// which trace surge limit kept the surge from raising sampling, while it is or was Limited.
	LimitReason string `json:"limitReason,omitempty"`

	// for a surge that ended at the maximum duration: when its service's metric went back to or
	// below the recovery threshold, after which the service can surge again.
	RecoveredAt *metav1.Time `json:"recoveredAt,omitempty"`

	// the processes that apply the surge's percentage, for each target.
	Targets []TraceSurgeTargetStatus `json:"targets,omitempty"`

	Timeline []TraceSurgeEvent `json:"timeline,omitempty"`
}

// TraceSurge records one surge of a sampling rule: a service's metric crossed the rule's threshold,
// the rule's percentage was raised for the workloads whose traces lead to the service, and
// restored once the metric recovered. It is kept in the status of the Sampling object that holds
// the rule while it is open and until its targets are confirmed back, or until its service
// recovers when it ended at the maximum duration; odigos insights keeps it after that.
type TraceSurge struct {
	// identifies the surge, e.g. 44136fa3-payments-tmfoef.
	Name string `json:"name"`

	// when the surge was recorded: its metric had crossed the threshold for the sustained window.
	StartedAt metav1.Time `json:"startedAt"`

	Spec   TraceSurgeSpec   `json:"spec"`
	Status TraceSurgeStatus `json:"status,omitempty"`
}

// Active reports whether the surge raises its targets' percentage.
func (s *TraceSurge) Active() bool {
	return s.Status.Phase == TraceSurgePhaseBoosting || s.Status.Phase == TraceSurgePhaseRecovering
}
