package podsmanifestinjectionstatus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	podsManifestInjection "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func podRequest(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: injectionNamespace, Name: name}}
}

// newReplicaSetOwnedPod builds a pod owned by a deployment's ReplicaSet, which is how the
// controller resolves a pod back to its workload.
func newReplicaSetOwnedPod(name, deploymentName, agentsHash string) *corev1.Pod {
	pod := withOwner(newInjectionPod(injectionNamespace, name, agentsHash),
		"apps/v1", "ReplicaSet", deploymentName+"-7d4c8b5f9b")
	return withPodLabel(pod, "app", deploymentName)
}

func TestPodsControllerReconcileTracksThePodAndSyncsItsWorkload(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		newReplicaSetOwnedPod("web-7d4c8b5f9b-abc", "web", injectionCurrentHash),
	)
	r := &PodsController{Client: c, PodsTracker: NewPodsTracker()}

	req := podRequest("web-7d4c8b5f9b-abc")
	result, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	pw, tracked := r.PodsTracker.GetPodWorkload(req)
	require.True(t, tracked, "the pod must be tracked so its deletion can be attributed later")
	assert.Equal(t, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}, pw)

	requireInjectionStatus(t, syncedInstrumentationConfig(t, ctx, c, "web",
		k8sconsts.WorkloadKindDeployment), true, false, false)
}

func TestPodsControllerReconcileIgnoresPodsWithoutAWorkloadOwner(t *testing.T) {
	ctx := newInjectionTestContext()
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		withPodLabel(newInjectionPod(injectionNamespace, "standalone", injectionStaleHash), "app", "web"),
	)
	r := &PodsController{Client: c, PodsTracker: NewPodsTracker()}

	req := podRequest("standalone")
	result, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	_, tracked := r.PodsTracker.GetPodWorkload(req)
	assert.False(t, tracked)
	assert.Nil(t, syncedInstrumentationConfig(t, ctx, c, "web",
		k8sconsts.WorkloadKindDeployment).Status.PodsManifestInjectionStatus)
}

// A deleted pod is already gone from the cluster, so the only way to know which workload status to
// recompute is the tracker. Without it the workload keeps reporting the deleted pod forever.
func TestPodsControllerReconcileResyncsAndUntracksADeletedPod(t *testing.T) {
	ctx := newInjectionTestContext()
	pod := newReplicaSetOwnedPod("web-7d4c8b5f9b-abc", "web", "")
	c := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
		pod,
	)
	r := &PodsController{Client: c, PodsTracker: NewPodsTracker()}

	req := podRequest("web-7d4c8b5f9b-abc")
	_, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	requireInjectionStatus(t, syncedInstrumentationConfig(t, ctx, c, "web",
		k8sconsts.WorkloadKindDeployment), false, false, true)

	require.NoError(t, c.Delete(ctx, pod))

	result, err := r.Reconcile(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	ic := syncedInstrumentationConfig(t, ctx, c, "web", k8sconsts.WorkloadKindDeployment)
	requireInjectionStatus(t, ic, false, false, false)
	assert.Equal(t, string(podsManifestInjection.PodsManifestInjectionReasonNoPods),
		injectionConditionReason(ic))

	_, tracked := r.PodsTracker.GetPodWorkload(req)
	assert.False(t, tracked, "a deleted pod must be untracked so the tracker does not grow forever")
}

// The controller watches every pod in the cluster, so deletions of pods it never tracked are
// common. Attempting to sync one would resolve to an empty workload and cost an API read per event.
func TestPodsControllerReconcileIgnoresAnUntrackedDeletedPod(t *testing.T) {
	ctx := newInjectionTestContext()
	gets := 0
	inner := newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
	)
	c := interceptor.NewClient(inner, interceptor.Funcs{
		Get: func(ctx context.Context, cl client.WithWatch, key client.ObjectKey, obj client.Object,
			opts ...client.GetOption) error {
			gets++
			return cl.Get(ctx, key, obj, opts...)
		},
	})
	r := &PodsController{Client: c, PodsTracker: NewPodsTracker()}

	result, err := r.Reconcile(ctx, podRequest("never-seen"))
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, 1, gets, "only the pod lookup itself may hit the API server")
	assert.Nil(t, syncedInstrumentationConfig(t, ctx, inner, "web",
		k8sconsts.WorkloadKindDeployment).Status.PodsManifestInjectionStatus)
}

// If the resync triggered by a deletion fails, the mapping has to survive: it is the only record
// of which workload the pod belonged to, and dropping it would strand the workload status.
func TestPodsControllerKeepsTrackingWhenTheDeletionResyncFails(t *testing.T) {
	ctx := newInjectionTestContext()
	listErr := apierrors.NewInternalError(errors.New("list pods failed"))
	c := interceptor.NewClient(newInjectionTestClient(t,
		newEffectiveConfigMap(""),
		newInjectionDeployment("web", map[string]string{"app": "web"}),
		newInjectionInstrumentationConfig("web", k8sconsts.WorkloadKindDeployment),
	), interceptor.Funcs{
		List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList,
			opts ...client.ListOption) error {
			if _, ok := list.(*corev1.PodList); ok {
				return listErr
			}
			return cl.List(ctx, list, opts...)
		},
	})

	r := &PodsController{Client: c, PodsTracker: NewPodsTracker()}
	req := podRequest("web-7d4c8b5f9b-abc")
	require.NoError(t, r.PodsTracker.SetPodWorkload(req, k8sconsts.PodWorkload{
		Namespace: injectionNamespace, Name: "web", Kind: k8sconsts.WorkloadKindDeployment,
	}))

	_, err := r.Reconcile(ctx, req)
	require.ErrorIs(t, err, listErr)

	_, tracked := r.PodsTracker.GetPodWorkload(req)
	assert.True(t, tracked, "the mapping must survive a failed resync so the retry can use it")
}

func TestHandleSyncError(t *testing.T) {
	genericErr := errors.New("boom")
	conflictErr := apierrors.NewConflict(
		schema.GroupResource{Group: "odigos.io", Resource: "instrumentationconfigs"}, "web",
		errors.New("conflict"))

	tests := []struct {
		name           string
		err            error
		expectedResult reconcile.Result
		expectedErr    error
	}{
		{
			name: "no error",
		},
		{
			// the effective config is written asynchronously at startup, so this is expected
			// during boot and must requeue instead of surfacing as a controller error
			name:           "the effective config is not available yet",
			err:            utils.ErrOdigosEffectiveConfigNotFound,
			expectedResult: reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Second},
		},
		{
			name:           "a conflicting status update requeues without an error",
			err:            conflictErr,
			expectedResult: reconcile.Result{Requeue: true},
		},
		{
			name: "a deleted object is ignored",
			err: apierrors.NewNotFound(
				schema.GroupResource{Group: "odigos.io", Resource: "instrumentationconfigs"}, "web"),
		},
		{
			name:        "any other error is surfaced",
			err:         genericErr,
			expectedErr: genericErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handleSyncError(tt.err)
			assert.Equal(t, tt.expectedResult, result)
			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.expectedErr)
			}
		})
	}
}
