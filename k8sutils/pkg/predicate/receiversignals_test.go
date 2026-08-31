package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func collectorsGroupWithSignals(signals ...common.ObservabilitySignal) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos-data-collection", Namespace: "odigos-system"},
		Status:     odigosv1.CollectorsGroupStatus{ReceiverSignals: signals},
	}
}

// Create is allowed for any non-nil object so that the controller sees the
// signals that already exist when it starts.
func TestReceiverSignalsChangedPredicateCreate(t *testing.T) {
	p := ReceiverSignalsChangedPredicate{}

	assert.True(t, p.Create(event.CreateEvent{Object: collectorsGroupWithSignals()}))
	assert.True(t, p.Create(event.CreateEvent{Object: collectorsGroupWithSignals(common.TracesObservabilitySignal)}))
	assert.False(t, p.Create(event.CreateEvent{Object: nil}))
}

func TestReceiverSignalsChangedPredicateUpdate(t *testing.T) {
	p := ReceiverSignalsChangedPredicate{}

	tests := []struct {
		name string
		old  []common.ObservabilitySignal
		new  []common.ObservabilitySignal
		want bool
	}{
		{
			name: "no signals on either side",
			old:  nil,
			new:  nil,
			want: false,
		},
		{
			name: "identical single signal",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
			new:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
			want: false,
		},
		{
			name: "identical multiple signals",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal},
			new:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal},
			want: false,
		},
		{
			name: "a signal was added",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
			new:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal},
			want: true,
		},
		{
			name: "a signal was removed",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal},
			new:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
			want: true,
		},
		{
			name: "first signal was replaced",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal},
			new:  []common.ObservabilitySignal{common.LogsObservabilitySignal, common.MetricsObservabilitySignal},
			want: true,
		},
		{
			// the comparison has to walk the whole slice, not just its first entry.
			name: "last signal was replaced",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal},
			new:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal},
			want: true,
		},
		{
			// same length and same set, but a different order still counts as a change.
			name: "signals were reordered",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal},
			new:  []common.ObservabilitySignal{common.MetricsObservabilitySignal, common.TracesObservabilitySignal},
			want: true,
		},
		{
			name: "signals appeared where there were none",
			old:  nil,
			new:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
			want: true,
		},
		{
			name: "all signals disappeared",
			old:  []common.ObservabilitySignal{common.TracesObservabilitySignal},
			new:  nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Update(event.UpdateEvent{
				ObjectOld: collectorsGroupWithSignals(tt.old...),
				ObjectNew: collectorsGroupWithSignals(tt.new...),
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReceiverSignalsChangedPredicateUpdateRejectsUnusableEvents(t *testing.T) {
	p := ReceiverSignalsChangedPredicate{}
	cg := collectorsGroupWithSignals(common.TracesObservabilitySignal)
	changed := collectorsGroupWithSignals(common.TracesObservabilitySignal, common.LogsObservabilitySignal)
	notACollectorsGroup := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "odigos-data-collection"}}

	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: nil, ObjectNew: changed}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: cg, ObjectNew: nil}))
	assert.False(t, p.Update(event.UpdateEvent{}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notACollectorsGroup, ObjectNew: changed}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: cg, ObjectNew: notACollectorsGroup}))
}

func TestReceiverSignalsChangedPredicateBlocksDeleteAndGeneric(t *testing.T) {
	p := ReceiverSignalsChangedPredicate{}
	cg := collectorsGroupWithSignals(common.TracesObservabilitySignal)

	assert.False(t, p.Delete(event.DeleteEvent{Object: cg}))
	assert.False(t, p.Generic(event.GenericEvent{Object: cg}))
}
