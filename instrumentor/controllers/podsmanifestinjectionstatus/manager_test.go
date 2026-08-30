package podsmanifestinjectionstatus

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func newPredicateInstrumentationConfig(agentsHash, rolloutHash string,
	rolloutCondition *metav1.Condition) *odigosv1.InstrumentationConfig {
	ic := &odigosv1.InstrumentationConfig{
		Spec:   odigosv1.InstrumentationConfigSpec{AgentsMetaHash: agentsHash},
		Status: odigosv1.InstrumentationConfigStatus{WorkloadRolloutHash: rolloutHash},
	}
	if rolloutCondition != nil {
		ic.Status.Conditions = []metav1.Condition{*rolloutCondition}
	}
	return ic
}

func rolloutCondition(reason odigosv1.WorkloadRolloutReason,
	status metav1.ConditionStatus) *metav1.Condition {
	return &metav1.Condition{
		Type:   odigosv1.WorkloadRolloutStatusConditionType,
		Status: status,
		Reason: string(reason),
	}
}

// The reported PodsManifestInjection reason is derived from the agents hash and from the rollout
// progress, so an update that changes either has to re-trigger the controller. An update that is
// filtered out here leaves a stale reason on the workload until some other event happens to fire.
func TestInstrumentationConfigPodsManifestInjectionPredicateUpdate(t *testing.T) {
	base := newPredicateInstrumentationConfig(injectionCurrentHash, injectionCurrentHash,
		rolloutCondition(odigosv1.WorkloadRolloutReasonTriggeredSuccessfully, metav1.ConditionTrue))

	tests := []struct {
		name     string
		old      client.Object
		new      client.Object
		expected bool
	}{
		{
			name:     "nothing relevant changed",
			old:      base.DeepCopy(),
			new:      base.DeepCopy(),
			expected: false,
		},
		{
			name: "the agents meta hash changed",
			old:  base.DeepCopy(),
			new: newPredicateInstrumentationConfig(injectionStaleHash, injectionCurrentHash,
				rolloutCondition(odigosv1.WorkloadRolloutReasonTriggeredSuccessfully, metav1.ConditionTrue)),
			expected: true,
		},
		{
			name: "the recorded rollout hash changed",
			old:  base.DeepCopy(),
			new: newPredicateInstrumentationConfig(injectionCurrentHash, injectionStaleHash,
				rolloutCondition(odigosv1.WorkloadRolloutReasonTriggeredSuccessfully, metav1.ConditionTrue)),
			expected: true,
		},
		{
			name:     "a rollout condition appeared",
			old:      newPredicateInstrumentationConfig(injectionCurrentHash, injectionCurrentHash, nil),
			new:      base.DeepCopy(),
			expected: true,
		},
		{
			name:     "the rollout condition disappeared",
			old:      base.DeepCopy(),
			new:      newPredicateInstrumentationConfig(injectionCurrentHash, injectionCurrentHash, nil),
			expected: true,
		},
		{
			name: "the rollout condition reason changed",
			old:  base.DeepCopy(),
			new: newPredicateInstrumentationConfig(injectionCurrentHash, injectionCurrentHash,
				rolloutCondition(odigosv1.WorkloadRolloutReasonFailedToPatch, metav1.ConditionTrue)),
			expected: true,
		},
		{
			name: "the rollout condition status changed",
			old:  base.DeepCopy(),
			new: newPredicateInstrumentationConfig(injectionCurrentHash, injectionCurrentHash,
				rolloutCondition(odigosv1.WorkloadRolloutReasonTriggeredSuccessfully, metav1.ConditionFalse)),
			expected: true,
		},
		{
			name: "only the rollout condition message changed",
			old:  base.DeepCopy(),
			new: func() client.Object {
				ic := base.DeepCopy()
				ic.Status.Conditions[0].Message = "a new message"
				return ic
			}(),
			expected: false,
		},
		{
			name:     "the objects are not instrumentation configs",
			old:      &corev1.Pod{},
			new:      &corev1.Pod{},
			expected: false,
		},
		{
			name:     "only the new object is an instrumentation config",
			old:      &corev1.Pod{},
			new:      base.DeepCopy(),
			expected: false,
		},
	}

	predicate := InstrumentationConfigPodsManifestInjectionPredicate{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, predicate.Update(event.UpdateEvent{
				ObjectOld: tt.old,
				ObjectNew: tt.new,
			}))
		})
	}
}

func TestInstrumentationConfigPodsManifestInjectionPredicateNonUpdateEvents(t *testing.T) {
	predicate := InstrumentationConfigPodsManifestInjectionPredicate{}

	// a newly created instrumentation config has no injection status yet, and a restarted
	// instrumentor rediscovers every workload through these create events
	assert.True(t, predicate.Create(event.CreateEvent{}))
	// the status lives on the instrumentation config, so there is nothing left to write
	assert.False(t, predicate.Delete(event.DeleteEvent{}))
	assert.True(t, predicate.Generic(event.GenericEvent{}))
}
