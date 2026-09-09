package instrumentednodes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// ****************
// syncNode() - stamping a node
// ****************

func TestSyncNode_StampsANodeRunningAPodWithInjectedAgents(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, time.Minute)

	require.NoError(t, err)
	assert.Zero(t, requeueAfter)
	stamped := requireNodeStamped(t, c, agentsNodeName)
	discoveredAt, parseErr := time.Parse(k8sconsts.FirstInstrumentedPodAtNodeLabelTimeFormat, stamped)
	require.NoError(t, parseErr, "label value %q must be readable back", stamped)
	assert.WithinDuration(t, time.Now().UTC(), discoveredAt, time.Minute)
}

func TestSyncNode_StampIsAValidLabelValue(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)

	require.NoError(t, firstErr(syncNode(ctx, c, agentsNodeName, 0)))

	stamped := requireNodeStamped(t, c, agentsNodeName)
	assert.Empty(t, validation.IsValidLabelValue(stamped),
		"the timestamp is stored as a label value, so it may not contain colons or other invalid characters")
}

// The label is written by syncFirstInstrumentedPodAtNodeLabel and read back by
// retentionRemaining; if the two ever disagree on the layout, the retention is
// silently skipped and the label is removed the moment the last pod is gone.
func TestSyncNode_StampIsUnderstoodByTheRetentionCheck(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)
	require.NoError(t, firstErr(syncNode(ctx, c, agentsNodeName, 0)))
	stamped := requireNodeStamped(t, c, agentsNodeName)

	remaining, delayRemoval := retentionRemaining(stamped, time.Hour)

	assert.True(t, delayRemoval, "a freshly written stamp must hold off removal")
	assert.Greater(t, remaining, 55*time.Minute)
	assert.LessOrEqual(t, remaining, time.Hour)
}

func TestSyncNode_KeepsTheOriginalStampWhileInstrumentedPodsRemain(t *testing.T) {
	ctx := instrumentedNodesContext()
	firstSeen := stampedAgo(3 * time.Hour)
	c := newInstrumentedNodesClient(t,
		nodeStampedAt(agentsNodeName, firstSeen),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, time.Minute)

	require.NoError(t, err)
	assert.Zero(t, requeueAfter)
	assert.Equal(t, firstSeen, requireNodeStamped(t, c, agentsNodeName),
		"the label records the FIRST discovery, so an already stamped node must not be re-stamped")
}

func TestSyncNode_IgnoresPodsScheduledOnOtherNodes(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		nodeNamed(otherNodeName),
		withAgentsInjected(podOnNode("instrumented", agentsNodeName)),
	)

	requeueAfter, err := syncNode(ctx, c, otherNodeName, time.Minute)

	require.NoError(t, err)
	assert.Zero(t, requeueAfter)
	requireNodeNotStamped(t, c, otherNodeName)
	requireNodeNotStamped(t, c, agentsNodeName)
}

func TestSyncNode_UnstampedNodeWithoutInstrumentedPodsIsANoOp(t *testing.T) {
	ctx := instrumentedNodesContext()
	node := nodeNamed(agentsNodeName)
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
				return errors.New("the node must not be patched")
			},
		},
		node,
		podOnNode("not-instrumented", agentsNodeName),
	)

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, time.Minute)

	require.NoError(t, err)
	assert.Zero(t, requeueAfter)
	requireNodeNotStamped(t, c, agentsNodeName)
}

// ****************
// syncNode() - removing the stamp
// ****************

func TestSyncNode_RemovesTheStampWhenNoInstrumentedPodsRemain(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeStampedAt(agentsNodeName, stampedAgo(time.Hour)),
		podOnNode("not-instrumented", agentsNodeName),
	)

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, 0)

	require.NoError(t, err)
	assert.Zero(t, requeueAfter)
	requireNodeNotStamped(t, c, agentsNodeName)

	var node corev1.Node
	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: agentsNodeName}, &node))
	assert.Equal(t, "linux", node.Labels["kubernetes.io/os"], "only the odigos label may be removed")
}

func TestSyncNode_KeepsTheStampUntilTheRetentionElapses(t *testing.T) {
	ctx := instrumentedNodesContext()
	stamp := stampedAgo(time.Minute)
	c := newInstrumentedNodesClient(t,
		nodeStampedAt(agentsNodeName, stamp),
		podOnNode("not-instrumented", agentsNodeName),
	)

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, 10*time.Minute)

	require.NoError(t, err)
	assert.Greater(t, requeueAfter, time.Duration(0), "removal must be retried once the retention elapses")
	assert.LessOrEqual(t, requeueAfter, 9*time.Minute)
	assert.Equal(t, stamp, requireNodeStamped(t, c, agentsNodeName))
}

func TestSyncNode_RemovesTheStampOnceTheRetentionElapsed(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeStampedAt(agentsNodeName, stampedAgo(11*time.Minute)),
		podOnNode("not-instrumented", agentsNodeName),
	)

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, 10*time.Minute)

	require.NoError(t, err)
	assert.Zero(t, requeueAfter)
	requireNodeNotStamped(t, c, agentsNodeName)
}

func TestSyncNode_RemovesAStampItCannotParse(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeStampedAt(agentsNodeName, "true"),
		podOnNode("not-instrumented", agentsNodeName),
	)

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, 10*time.Minute)

	require.NoError(t, err)
	assert.Zero(t, requeueAfter)
	requireNodeNotStamped(t, c, agentsNodeName)
}

// ****************
// syncNode() - error paths
// ****************

func TestSyncNode_MissingNodeIsNotAnError(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t, withAgentsInjected(podOnNode("instrumented", agentsNodeName)))

	requeueAfter, err := syncNode(ctx, c, agentsNodeName, time.Minute)

	assert.NoError(t, err)
	assert.Zero(t, requeueAfter)
}

func TestSyncNode_NodeGetErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return apierrors.NewInternalError(errors.New("etcd is down"))
			},
		},
		nodeNamed(agentsNodeName),
	)

	_, err := syncNode(ctx, c, agentsNodeName, time.Minute)

	assert.ErrorContains(t, err, "etcd is down")
}

func TestSyncNode_PodListErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
				return errors.New("pod cache is not synced")
			},
		},
		nodeNamed(agentsNodeName),
	)

	_, err := syncNode(ctx, c, agentsNodeName, time.Minute)

	assert.ErrorContains(t, err, "pod cache is not synced")
	requireNodeNotStamped(t, c, agentsNodeName)
}

func TestSyncNode_PatchErrorIsPropagated(t *testing.T) {
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

	_, err := syncNode(ctx, c, agentsNodeName, time.Minute)

	assert.ErrorContains(t, err, "no permission to label nodes")
}

// ****************
// hasInstrumentedPod() tests
// ****************

func TestHasInstrumentedPod_ReportsAPodWhoseWorkloadIsMarkedForInstrumentation(t *testing.T) {
	ctx := instrumentedNodesContext()
	// no agents label: the pod predates the source, or was instrumented without a restart
	c := newInstrumentedNodesClient(t,
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		instrumentationConfig(checkoutConfigName, instrumentedNodesNamespace),
	)

	instrumented, err := hasInstrumentedPod(ctx, c, agentsNodeName)

	require.NoError(t, err)
	assert.True(t, instrumented)
}

func TestHasInstrumentedPod_IgnoresAWorkloadThatIsNotMarkedForInstrumentation(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		// an instrumentation config for a different workload in the same namespace
		instrumentationConfig("deployment-frontend", instrumentedNodesNamespace),
	)

	instrumented, err := hasInstrumentedPod(ctx, c, agentsNodeName)

	require.NoError(t, err)
	assert.False(t, instrumented)
}

func TestHasInstrumentedPod_IgnoresAnInstrumentationConfigInAnotherNamespace(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		instrumentationConfig(checkoutConfigName, instrumentedNodesOtherNamespace),
	)

	instrumented, err := hasInstrumentedPod(ctx, c, agentsNodeName)

	require.NoError(t, err)
	assert.False(t, instrumented)
}

func TestHasInstrumentedPod_IgnoresAPodThatBelongsToNoWorkload(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t, podOnNode("standalone", agentsNodeName))

	instrumented, err := hasInstrumentedPod(ctx, c, agentsNodeName)

	require.NoError(t, err)
	assert.False(t, instrumented)
}

// The agents label is checked for every pod before any instrumentation config is
// fetched, so a node full of pods costs a single list in the common case.
func TestHasInstrumentedPod_AgentsLabelIsCheckedBeforeAnyInstrumentationConfigLookup(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
				if _, isConfig := obj.(*odigosv1.InstrumentationConfig); isConfig {
					return errors.New("no instrumentation config should be fetched")
				}
				return nil
			},
		},
		// sorts before the pod carrying the label, so a single-pass implementation
		// would look its instrumentation config up first
		deploymentPod("checkout-7d4c8b5f9b-aaaaa", agentsNodeName),
		withAgentsInjected(deploymentPod("checkout-7d4c8b5f9b-zzzzz", agentsNodeName)),
	)

	instrumented, err := hasInstrumentedPod(ctx, c, agentsNodeName)

	require.NoError(t, err)
	assert.True(t, instrumented)
}

func TestHasInstrumentedPod_InstrumentationConfigGetErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
				if _, isConfig := obj.(*odigosv1.InstrumentationConfig); isConfig {
					return apierrors.NewInternalError(errors.New("instrumentation config cache is cold"))
				}
				return nil
			},
		},
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
	)

	instrumented, err := hasInstrumentedPod(ctx, c, agentsNodeName)

	assert.ErrorContains(t, err, "instrumentation config cache is cold")
	assert.False(t, instrumented)
}

// ****************
// podHasInstrumentationConfig() tests
// ****************

func TestPodHasInstrumentationConfig_LooksTheConfigUpByTheWorkloadOfThePod(t *testing.T) {
	ctx := instrumentedNodesContext()
	// the name is spelled out rather than derived, because it is the contract with
	// whoever creates the instrumentation config
	c := newInstrumentedNodesClient(t, instrumentationConfig("deployment-checkout", instrumentedNodesNamespace))

	found, err := podHasInstrumentationConfig(ctx, c, deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName))

	require.NoError(t, err)
	assert.True(t, found)
}

func TestPodHasInstrumentationConfig_CronJobPodResolvesToItsCronJob(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t, instrumentationConfig("cronjob-reports", instrumentedNodesNamespace))

	found, err := podHasInstrumentationConfig(ctx, c, cronJobPod("reports-28901234-abcde", agentsNodeName))

	require.NoError(t, err)
	assert.True(t, found)
}

func TestPodHasInstrumentationConfig_PodWithAnUnusableOwnerIsAnError(t *testing.T) {
	ctx := instrumentedNodesContext()
	// owned by a Node but not a static pod: odigos cannot resolve a workload for it
	pod := podOnNode("orphan", agentsNodeName)
	pod.OwnerReferences = []metav1.OwnerReference{{APIVersion: "v1", Kind: "Node", Name: agentsNodeName}}
	c := newInstrumentedNodesClient(t)

	found, err := podHasInstrumentationConfig(ctx, c, pod)

	assert.Error(t, err)
	assert.False(t, found)
}

// ****************
// retentionRemaining() tests
// ****************

func TestRetentionRemaining(t *testing.T) {
	tests := []struct {
		name         string
		labelValue   string
		retention    time.Duration
		delayRemoval bool
		atLeast      time.Duration
		atMost       time.Duration
	}{
		{
			name:         "no retention configured removes the label right away",
			labelValue:   stampedAgo(0),
			retention:    0,
			delayRemoval: false,
		},
		{
			name:         "a negative retention removes the label right away",
			labelValue:   stampedAgo(0),
			retention:    -time.Minute,
			delayRemoval: false,
		},
		{
			name:         "a label value that is not a timestamp is removed right away",
			labelValue:   "true",
			retention:    time.Hour,
			delayRemoval: false,
		},
		{
			name:         "an empty label value is removed right away",
			labelValue:   "",
			retention:    time.Hour,
			delayRemoval: false,
		},
		{
			name:         "a fresh label waits out the whole retention",
			labelValue:   stampedAgo(0),
			retention:    time.Hour,
			delayRemoval: true,
			atLeast:      59 * time.Minute,
			atMost:       time.Hour,
		},
		{
			name:         "a label waits out only what is left of the retention",
			labelValue:   stampedAgo(50 * time.Minute),
			retention:    time.Hour,
			delayRemoval: true,
			atLeast:      9 * time.Minute,
			atMost:       10 * time.Minute,
		},
		{
			name:         "a label as old as the retention is removed",
			labelValue:   stampedAgo(time.Hour),
			retention:    time.Hour,
			delayRemoval: false,
		},
		{
			name:         "a label older than the retention is removed",
			labelValue:   stampedAgo(2 * time.Hour),
			retention:    time.Hour,
			delayRemoval: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining, delayRemoval := retentionRemaining(tt.labelValue, tt.retention)

			assert.Equal(t, tt.delayRemoval, delayRemoval)
			if !tt.delayRemoval {
				assert.Zero(t, remaining)
				return
			}
			assert.Greater(t, remaining, tt.atLeast)
			assert.LessOrEqual(t, remaining, tt.atMost)
		})
	}
}
