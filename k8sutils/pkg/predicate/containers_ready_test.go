package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func startedPtr(b bool) *bool {
	return &b
}

// readinessPod builds a running pod with a single container whose readiness is
// controlled by `ready`. A ready container is also reported as started, which is
// what the kubelet does for long running pods.
func readinessPod(ready bool) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: ready, Started: startedPtr(ready)},
			},
		},
	}
}

func TestAllContainersBecomeReadyPredicateUpdate(t *testing.T) {
	p := &AllContainersBecomeReadyPredicate{}

	tests := []struct {
		name     string
		oldReady bool
		newReady bool
		want     bool
	}{
		{name: "containers become ready", oldReady: false, newReady: true, want: true},
		{name: "containers were already ready", oldReady: true, newReady: true, want: false},
		{name: "containers are still not ready", oldReady: false, newReady: false, want: false},
		{name: "containers stopped being ready", oldReady: true, newReady: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Update(event.UpdateEvent{
				ObjectOld: readinessPod(tt.oldReady),
				ObjectNew: readinessPod(tt.newReady),
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

// A pod that is terminating must not be treated as newly ready, or the consumer
// would start inspecting a container that is about to disappear.
func TestAllContainersBecomeReadyPredicateSkipsDeletingPods(t *testing.T) {
	p := &AllContainersBecomeReadyPredicate{}
	deletedAt := metav1.Now()

	deletingPod := readinessPod(true)
	deletingPod.DeletionTimestamp = &deletedAt

	got := p.Update(event.UpdateEvent{
		ObjectOld: readinessPod(false),
		ObjectNew: deletingPod,
	})
	assert.False(t, got)
}

// The pod's containers must all be ready, not just the first one.
func TestAllContainersBecomeReadyPredicateRequiresEveryContainer(t *testing.T) {
	p := &AllContainersBecomeReadyPredicate{}

	partiallyReady := readinessPod(true)
	partiallyReady.Status.ContainerStatuses = append(partiallyReady.Status.ContainerStatuses,
		corev1.ContainerStatus{Name: "sidecar", Ready: false, Started: startedPtr(true)})

	allReady := readinessPod(true)
	allReady.Status.ContainerStatuses = append(allReady.Status.ContainerStatuses,
		corev1.ContainerStatus{Name: "sidecar", Ready: true, Started: startedPtr(true)})

	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: readinessPod(false), ObjectNew: partiallyReady}))
	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: partiallyReady, ObjectNew: allReady}))
}

func TestAllContainersBecomeReadyPredicateUpdateRejectsUnusableEvents(t *testing.T) {
	p := &AllContainersBecomeReadyPredicate{}
	notReady := readinessPod(false)
	ready := readinessPod(true)
	notAPod := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "app-pod"}}

	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: nil, ObjectNew: ready}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notReady, ObjectNew: nil}))
	assert.False(t, p.Update(event.UpdateEvent{}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notAPod, ObjectNew: ready}))
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: notReady, ObjectNew: notAPod}))
}

// Only the readiness transition is interesting; a create event for an
// already-ready pod is handled by the controller's own startup logic.
func TestAllContainersBecomeReadyPredicateBlocksCreateDeleteAndGeneric(t *testing.T) {
	p := &AllContainersBecomeReadyPredicate{}
	ready := readinessPod(true)

	assert.False(t, p.Create(event.CreateEvent{Object: ready}))
	assert.False(t, p.Delete(event.DeleteEvent{Object: ready}))
	assert.False(t, p.Generic(event.GenericEvent{Object: ready}))
}
