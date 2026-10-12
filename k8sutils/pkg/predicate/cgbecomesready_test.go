package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func collectorsGroup(name string, ready bool) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system"},
		Status:     odigosv1.CollectorsGroupStatus{Ready: ready},
	}
}

func TestCgBecomesReadyPredicateCreate(t *testing.T) {
	p := &CgBecomesReadyPredicate{}

	tests := []struct {
		name   string
		object *odigosv1.CollectorsGroup
		want   bool
	}{
		{name: "ready collectors group", object: collectorsGroup("odigos-gateway", true), want: true},
		{name: "not ready collectors group", object: collectorsGroup("odigos-gateway", false), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, p.Create(event.CreateEvent{Object: tt.object}))
		})
	}

	t.Run("nil object", func(t *testing.T) {
		assert.False(t, p.Create(event.CreateEvent{Object: nil}))
	})

	t.Run("object that is not a collectors group", func(t *testing.T) {
		other := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "odigos-gateway"}}
		assert.False(t, p.Create(event.CreateEvent{Object: other}))
	})
}

// Only the not-ready to ready transition may pass. Controllers using this filter
// do one-shot work when the collectors are first able to receive data, so letting
// a steady-state ready update through would make them reconcile on every status write.
func TestCgBecomesReadyPredicateUpdate(t *testing.T) {
	p := &CgBecomesReadyPredicate{}

	tests := []struct {
		name     string
		wasReady bool
		nowReady bool
		want     bool
	}{
		{name: "becomes ready", wasReady: false, nowReady: true, want: true},
		{name: "stays ready", wasReady: true, nowReady: true, want: false},
		{name: "stays not ready", wasReady: false, nowReady: false, want: false},
		{name: "becomes not ready", wasReady: true, nowReady: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Update(event.UpdateEvent{
				ObjectOld: collectorsGroup("odigos-gateway", tt.wasReady),
				ObjectNew: collectorsGroup("odigos-gateway", tt.nowReady),
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCgBecomesReadyPredicateUpdateRejectsUnusableEvents(t *testing.T) {
	p := &CgBecomesReadyPredicate{}
	notReady := collectorsGroup("odigos-gateway", false)
	ready := collectorsGroup("odigos-gateway", true)
	notACollectorsGroup := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "odigos-gateway"}}

	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: nil, ObjectNew: ready}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notReady, ObjectNew: nil}))
	assert.False(t, p.Update(event.UpdateEvent{}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notACollectorsGroup, ObjectNew: ready}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notReady, ObjectNew: notACollectorsGroup}))
}

func TestCgBecomesReadyPredicateBlocksDeleteAndGeneric(t *testing.T) {
	p := &CgBecomesReadyPredicate{}
	ready := collectorsGroup("odigos-gateway", true)

	assert.False(t, p.Delete(event.DeleteEvent{Object: ready}))
	assert.False(t, p.Generic(event.GenericEvent{Object: ready}))
}
