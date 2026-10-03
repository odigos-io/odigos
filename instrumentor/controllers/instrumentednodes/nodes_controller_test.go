package instrumentednodes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var _ predicate.Predicate = nodeInstrumentedPodsLabelPredicate{}

// nodesPeriodicResync is the interval the nodes controller re-syncs at; it is
// the only thing that eventually removes a label after pods are deleted.
const nodesPeriodicResync = time.Minute

// ****************
// nodeInstrumentedPodsLabelPredicate tests
// ****************

func TestNodeInstrumentedPodsLabelPredicate_Create(t *testing.T) {
	p := nodeInstrumentedPodsLabelPredicate{}

	// pods can be in the cache before their node is, so every new node is synced
	assert.True(t, p.Create(event.CreateEvent{Object: nodeNamed(agentsNodeName)}))
	assert.True(t, p.Create(event.CreateEvent{Object: nodeStampedAt(agentsNodeName, stampedAgo(0))}))
}

func TestNodeInstrumentedPodsLabelPredicate_Update(t *testing.T) {
	p := nodeInstrumentedPodsLabelPredicate{}
	stamp := stampedAgo(time.Hour)

	tests := []struct {
		name      string
		old       client.Object
		updated   client.Object
		reconcile bool
	}{
		{
			name:      "the label was added",
			old:       nodeNamed(agentsNodeName),
			updated:   nodeStampedAt(agentsNodeName, stamp),
			reconcile: true,
		},
		{
			name:      "the label was removed",
			old:       nodeStampedAt(agentsNodeName, stamp),
			updated:   nodeNamed(agentsNodeName),
			reconcile: true,
		},
		{
			name:      "the label value changed",
			old:       nodeStampedAt(agentsNodeName, stamp),
			updated:   nodeStampedAt(agentsNodeName, stampedAgo(0)),
			reconcile: true,
		},
		{
			name:      "an update that does not touch the label",
			old:       nodeStampedAt(agentsNodeName, stamp),
			updated:   nodeStampedAt(agentsNodeName, stamp),
			reconcile: false,
		},
		{
			name:      "the old object is not a node",
			old:       podOnNode("pod", agentsNodeName),
			updated:   nodeStampedAt(agentsNodeName, stamp),
			reconcile: false,
		},
		{
			name:      "the new object is not a node",
			old:       nodeStampedAt(agentsNodeName, stamp),
			updated:   podOnNode("pod", agentsNodeName),
			reconcile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.reconcile, p.Update(event.UpdateEvent{ObjectOld: tt.old, ObjectNew: tt.updated}))
		})
	}
}

func TestNodeInstrumentedPodsLabelPredicate_IgnoresUnrelatedLabelChanges(t *testing.T) {
	p := nodeInstrumentedPodsLabelPredicate{}
	old := nodeNamed(agentsNodeName)
	updated := nodeNamed(agentsNodeName)
	updated.Labels["topology.kubernetes.io/zone"] = "us-east-1a"

	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: old, ObjectNew: updated}))
}

func TestNodeInstrumentedPodsLabelPredicate_IgnoresDeletesAndGenericEvents(t *testing.T) {
	p := nodeInstrumentedPodsLabelPredicate{}

	assert.False(t, p.Delete(event.DeleteEvent{Object: nodeStampedAt(agentsNodeName, stampedAgo(0))}))
	assert.False(t, p.Generic(event.GenericEvent{Object: nodeStampedAt(agentsNodeName, stampedAgo(0))}))
}

// ****************
// NodesReconciler tests
// ****************

func nodeRequest(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: name}}
}

func TestNodesReconciler_StampsTheNodeAndKeepsResyncing(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)
	r := &NodesReconciler{Client: c, NodeLabelRetention: 0}

	result, err := r.Reconcile(ctx, nodeRequest(agentsNodeName))

	require.NoError(t, err)
	requireNodeStamped(t, c, agentsNodeName)
	assert.Equal(t, nodesPeriodicResync, result.RequeueAfter,
		"pod deletions are not watched, so the node must be re-synced periodically")
}

func TestNodesReconciler_RemovesTheStampOfANodeWithoutInstrumentedPods(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t, nodeStampedAt(agentsNodeName, stampedAgo(time.Hour)))
	r := &NodesReconciler{Client: c, NodeLabelRetention: 0}

	result, err := r.Reconcile(ctx, nodeRequest(agentsNodeName))

	require.NoError(t, err)
	requireNodeNotStamped(t, c, agentsNodeName)
	assert.Equal(t, nodesPeriodicResync, result.RequeueAfter)
}

func TestNodesReconciler_DeletedNodeStillResyncs(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t)
	r := &NodesReconciler{Client: c, NodeLabelRetention: 0}

	result, err := r.Reconcile(ctx, nodeRequest("already-deleted"))

	require.NoError(t, err)
	assert.Equal(t, nodesPeriodicResync, result.RequeueAfter)
}

func TestNodesReconciler_RetriesAsSoonAsTheRetentionExpires(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t, nodeStampedAt(agentsNodeName, stampedAgo(time.Minute)))
	r := &NodesReconciler{Client: c, NodeLabelRetention: 90 * time.Second}

	result, err := r.Reconcile(ctx, nodeRequest(agentsNodeName))

	require.NoError(t, err)
	assert.Greater(t, result.RequeueAfter, time.Duration(0))
	assert.Less(t, result.RequeueAfter, nodesPeriodicResync,
		"a retention that expires before the next resync must be requeued for its own deadline")
	assert.NotEmpty(t, requireNodeStamped(t, c, agentsNodeName))
}

func TestNodesReconciler_ALongRetentionDoesNotDelayTheResync(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t, nodeStampedAt(agentsNodeName, stampedAgo(time.Minute)))
	r := &NodesReconciler{Client: c, NodeLabelRetention: time.Hour}

	result, err := r.Reconcile(ctx, nodeRequest(agentsNodeName))

	require.NoError(t, err)
	assert.Equal(t, nodesPeriodicResync, result.RequeueAfter)
	assert.NotEmpty(t, requireNodeStamped(t, c, agentsNodeName))
}

func TestNodesReconciler_ConflictIsRetriedWithoutAnError(t *testing.T) {
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
	r := &NodesReconciler{Client: c, NodeLabelRetention: 0}

	result, err := r.Reconcile(ctx, nodeRequest(agentsNodeName))

	require.NoError(t, err)
	assert.True(t, result.Requeue)
	assert.Zero(t, result.RequeueAfter)
}

func TestNodesReconciler_ErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
				return errors.New("pod cache is not synced")
			},
		},
		nodeNamed(agentsNodeName),
	)
	r := &NodesReconciler{Client: c, NodeLabelRetention: 0}

	result, err := r.Reconcile(ctx, nodeRequest(agentsNodeName))

	assert.ErrorContains(t, err, "pod cache is not synced")
	assert.Zero(t, result.RequeueAfter)
}

// The periodic resync is what makes the label eventually disappear, so it must
// not be shortened by a node that has nothing to do.
func TestNodesReconciler_ResyncsEveryNodeAtTheSameInterval(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		nodeNamed(otherNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)
	r := &NodesReconciler{Client: c, NodeLabelRetention: time.Hour}

	stamped, err := r.Reconcile(ctx, nodeRequest(agentsNodeName))
	require.NoError(t, err)
	idle, err := r.Reconcile(ctx, nodeRequest(otherNodeName))
	require.NoError(t, err)

	assert.Equal(t, nodesPeriodicResync, stamped.RequeueAfter)
	assert.Equal(t, nodesPeriodicResync, idle.RequeueAfter)
	requireNodeNotStamped(t, c, otherNodeName)
	assert.NotEmpty(t, requireNodeStamped(t, c, agentsNodeName))
}

// The odiglet daemonset selects nodes by this label in the helm chart, which
// cannot reference the constant, so its value is part of the public contract.
func TestFirstInstrumentedPodAtNodeLabelIsTheLabelTheOdigletSelectsOn(t *testing.T) {
	assert.Equal(t, "odigos.io/first-instrumented-pod-at", k8sconsts.FirstInstrumentedPodAtNodeLabel)
}
