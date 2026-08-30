package podsmanifestinjectionstatus

import (
	"fmt"
	"sync"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestPodsTrackerRoundTrip(t *testing.T) {
	tracker := NewPodsTracker()
	req := podRequest("web-7d4c8b5f9b-abc")
	pw := k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}

	_, ok := tracker.GetPodWorkload(req)
	assert.False(t, ok)

	require.NoError(t, tracker.SetPodWorkload(req, pw))
	stored, ok := tracker.GetPodWorkload(req)
	require.True(t, ok)
	assert.Equal(t, pw, stored)

	// a pod recreated under a renamed workload must overwrite the previous mapping
	renamed := k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web-v2", Kind: k8sconsts.WorkloadKindDeployment,
	}
	require.NoError(t, tracker.SetPodWorkload(req, renamed))
	stored, ok = tracker.GetPodWorkload(req)
	require.True(t, ok)
	assert.Equal(t, renamed, stored)

	tracker.DeletePodWorkload(req)
	_, ok = tracker.GetPodWorkload(req)
	assert.False(t, ok)

	// deleting an absent mapping is a no-op, since a pod can be untracked more than once
	tracker.DeletePodWorkload(req)
}

// The tracker holds one entry per live pod and is only pruned on pod deletion events, so the cap
// is what stops a missed deletion from growing the map without bound.
func TestPodsTrackerRefusesToGrowPastItsMaxSize(t *testing.T) {
	tracker := NewPodsTracker()
	pw := k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}

	var lastAccepted ctrl.Request
	for i := 0; i < maxPodsTrackerSize; i++ {
		lastAccepted = podRequest(fmt.Sprintf("pod-%d", i))
		require.NoError(t, tracker.SetPodWorkload(lastAccepted, pw))
	}

	overflow := podRequest("pod-overflow")
	require.Error(t, tracker.SetPodWorkload(overflow, pw))
	_, ok := tracker.GetPodWorkload(overflow)
	assert.False(t, ok, "a rejected mapping must not be stored")

	// freeing a slot lets the tracker accept new pods again
	tracker.DeletePodWorkload(lastAccepted)
	require.NoError(t, tracker.SetPodWorkload(overflow, pw))
	_, ok = tracker.GetPodWorkload(overflow)
	assert.True(t, ok)
}

// Pod events are reconciled concurrently, so every tracker access has to be serialized.
func TestPodsTrackerConcurrentAccess(t *testing.T) {
	tracker := NewPodsTracker()
	const workers = 16

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				req := ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: injectionNamespace, Name: fmt.Sprintf("pod-%d-%d", w, i),
				}}
				_ = tracker.SetPodWorkload(req, k8sconsts.PodWorkload{
					Namespace: injectionNamespace,
					Name:      fmt.Sprintf("web-%d", w),
					Kind:      k8sconsts.WorkloadKindDeployment,
				})
				tracker.GetPodWorkload(req)
				tracker.DeletePodWorkload(req)
			}
		}(w)
	}
	wg.Wait()

	_, ok := tracker.GetPodWorkload(podRequest("pod-0-0"))
	assert.False(t, ok, "every mapping written by the workers was deleted again")
}
