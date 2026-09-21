package instrumentednodes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ predicate.Predicate = podNodeNamePredicate{}

// ****************
// podNodeNamePredicate tests
// ****************

func TestPodNodeNamePredicate_Create(t *testing.T) {
	p := podNodeNamePredicate{}

	assert.True(t, p.Create(event.CreateEvent{Object: podOnNode("scheduled", agentsNodeName)}),
		"a pod that is already scheduled may be the first instrumented pod on its node")
	assert.False(t, p.Create(event.CreateEvent{Object: podOnNode("pending", "")}),
		"an unscheduled pod has no node to label")
	assert.False(t, p.Create(event.CreateEvent{Object: nodeNamed(agentsNodeName)}))
}

func TestPodNodeNamePredicate_Update(t *testing.T) {
	p := podNodeNamePredicate{}

	tests := []struct {
		name      string
		old       client.Object
		updated   client.Object
		reconcile bool
	}{
		{
			name:      "a pod that just got scheduled",
			old:       podOnNode("pod", ""),
			updated:   podOnNode("pod", agentsNodeName),
			reconcile: true,
		},
		{
			name:      "an update that does not touch the node",
			old:       podOnNode("pod", agentsNodeName),
			updated:   withAgentsInjected(podOnNode("pod", agentsNodeName)),
			reconcile: false,
		},
		{
			name:      "the old object is not a pod",
			old:       nodeNamed(agentsNodeName),
			updated:   podOnNode("pod", agentsNodeName),
			reconcile: false,
		},
		{
			name:      "the new object is not a pod",
			old:       podOnNode("pod", agentsNodeName),
			updated:   nodeNamed(agentsNodeName),
			reconcile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.reconcile, p.Update(event.UpdateEvent{ObjectOld: tt.old, ObjectNew: tt.updated}))
		})
	}
}

func TestPodNodeNamePredicate_IgnoresDeletesAndGenericEvents(t *testing.T) {
	p := podNodeNamePredicate{}

	// a deleted pod carries its node name, but the label is removed by the
	// periodic node resync rather than here
	assert.False(t, p.Delete(event.DeleteEvent{Object: withAgentsInjected(podOnNode("gone", agentsNodeName))}))
	assert.False(t, p.Generic(event.GenericEvent{Object: withAgentsInjected(podOnNode("pod", agentsNodeName))}))
}

// ****************
// PodsReconciler tests
// ****************

func podRequest(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: instrumentedNodesNamespace, Name: name}}
}

func TestPodsReconciler_StampsTheNodeOfAPodWithInjectedAgents(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)
	r := &PodsReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, podRequest("instrumented"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	requireNodeStamped(t, c, agentsNodeName)
}

func TestPodsReconciler_IgnoresAPodWithoutInjectedAgents(t *testing.T) {
	ctx := instrumentedNodesContext()
	// the pod belongs to an instrumented workload, but its manifest has no agents:
	// stamping the node for it is the instrumentation config controller's job
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		instrumentationConfig(checkoutConfigName, instrumentedNodesNamespace),
	)
	r := &PodsReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, podRequest("checkout-7d4c8b5f9b-p2xzq"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	requireNodeNotStamped(t, c, agentsNodeName)
}

func TestPodsReconciler_IgnoresAnUnscheduledPod(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Get: func(ctx context.Context, inner client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, isNode := obj.(*corev1.Node); isNode {
					return errors.New("a pod with no node must not start a node sync")
				}
				return inner.Get(ctx, key, obj, opts...)
			},
		},
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("pending", "")),
	)
	r := &PodsReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, podRequest("pending"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestPodsReconciler_DeletedPodIsANoOp(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t, nodeNamed(agentsNodeName))
	r := &PodsReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, podRequest("already-deleted"))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	requireNodeNotStamped(t, c, agentsNodeName)
}

func TestPodsReconciler_PodGetErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return apierrors.NewInternalError(errors.New("pod informer is not synced"))
			},
		},
		nodeNamed(agentsNodeName),
	)
	r := &PodsReconciler{Client: c, NodeLabelRetention: time.Minute}

	_, err := r.Reconcile(ctx, podRequest("instrumented"))

	assert.ErrorContains(t, err, "pod informer is not synced")
}

func TestPodsReconciler_ConflictOnTheNodeIsRetriedWithoutAnError(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
				return apierrors.NewConflict(schema.GroupResource{Resource: "nodes"}, agentsNodeName, errors.New("the object has been modified"))
			},
		},
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)
	r := &PodsReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, podRequest("instrumented"))

	require.NoError(t, err, "a conflict is an expected race, not an error to log")
	assert.True(t, result.Requeue)
}

func TestPodsReconciler_NodePatchErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
				return apierrors.NewForbidden(schema.GroupResource{Resource: "nodes"}, agentsNodeName, errors.New("no permission to label nodes"))
			},
		},
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)
	r := &PodsReconciler{Client: c, NodeLabelRetention: time.Minute}

	_, err := r.Reconcile(ctx, podRequest("instrumented"))

	assert.ErrorContains(t, err, "no permission to label nodes")
}

// A pod event can be served from a cache that is ahead of the pod list index, in
// which case the node has no instrumented pods left and its label is waiting out
// the retention. The reconcile has to carry that deadline back to the queue.
func TestPodsReconciler_CarriesTheRetentionDeadlineBackToTheQueue(t *testing.T) {
	ctx := instrumentedNodesContext()
	staleCachedPod := withAgentsInjected(podOnNode("terminated", agentsNodeName))
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Get: func(ctx context.Context, inner client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if pod, isPod := obj.(*corev1.Pod); isPod {
					staleCachedPod.DeepCopyInto(pod)
					return nil
				}
				return inner.Get(ctx, key, obj, opts...)
			},
		},
		nodeStampedAt(agentsNodeName, stampedAgo(time.Minute)),
	)
	r := &PodsReconciler{Client: c, NodeLabelRetention: 10 * time.Minute}

	result, err := r.Reconcile(ctx, podRequest("terminated"))

	require.NoError(t, err)
	assert.Greater(t, result.RequeueAfter, time.Duration(0))
	assert.LessOrEqual(t, result.RequeueAfter, 9*time.Minute)
	requireNodeStamped(t, c, agentsNodeName)
}
