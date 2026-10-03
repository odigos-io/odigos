package instrumentednodes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func instrumentationConfigRequest(name string) ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: instrumentedNodesNamespace, Name: name}}
}

// ****************
// InstrumentationConfigReconciler tests
// ****************

// A workload marked for instrumentation has to stamp its nodes before any agent
// is injected, so odiglet is there to run runtime detection.
func TestInstrumentationConfigReconciler_StampsEveryNodeRunningTheWorkload(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		nodeNamed(otherNodeName),
		checkoutDeployment(),
		instrumentationConfig(checkoutConfigName, instrumentedNodesNamespace),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		deploymentPod("checkout-7d4c8b5f9b-h9vlk", otherNodeName),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, instrumentationConfigRequest(checkoutConfigName))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	requireNodeStamped(t, c, agentsNodeName)
	requireNodeStamped(t, c, otherNodeName)
}

func TestInstrumentationConfigReconciler_IgnoresPodsOfOtherWorkloadsAndNamespaces(t *testing.T) {
	ctx := instrumentedNodesContext()
	otherWorkloadPod := podOnNode("frontend-6bc4f8d5c-xk2mn", otherNodeName)
	otherWorkloadPod.Labels = map[string]string{"app": "frontend"}
	sameSelectorOtherNamespace := deploymentPod("checkout-7d4c8b5f9b-elsewhere", otherNodeName)
	sameSelectorOtherNamespace.Namespace = instrumentedNodesOtherNamespace

	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		nodeNamed(otherNodeName),
		checkoutDeployment(),
		instrumentationConfig(checkoutConfigName, instrumentedNodesNamespace),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		otherWorkloadPod,
		sameSelectorOtherNamespace,
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	_, err := r.Reconcile(ctx, instrumentationConfigRequest(checkoutConfigName))

	require.NoError(t, err)
	requireNodeStamped(t, c, agentsNodeName)
	requireNodeNotStamped(t, c, otherNodeName)
}

func TestInstrumentationConfigReconciler_StaticPodStampsItsOwnNode(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		staticPod("kube-apiserver", agentsNodeName),
		instrumentationConfig("staticpod-kube-apiserver", instrumentedNodesNamespace),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	_, err := r.Reconcile(ctx, instrumentationConfigRequest("staticpod-kube-apiserver"))

	require.NoError(t, err)
	requireNodeStamped(t, c, agentsNodeName)
}

// Cron job pods carry no label selector of their own, so they are matched
// through the job that owns them.
func TestInstrumentationConfigReconciler_CronJobMatchesItsPodsByOwnerReference(t *testing.T) {
	ctx := instrumentedNodesContext()
	otherCronJobPod := podOnNode("cleanup-28901234-abcde", otherNodeName)
	otherCronJobPod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "batch/v1",
		Kind:       "Job",
		Name:       "cleanup-28901234",
	}}
	// a deployment that happens to share the name of the cron job
	sameNameOtherKind := podOnNode("reports-7d4c8b5f9b-w4trp", otherNodeName)
	sameNameOtherKind.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "apps/v1",
		Kind:       "ReplicaSet",
		Name:       "reports-7d4c8b5f9b",
	}}

	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		nodeNamed(otherNodeName),
		reportsCronJobObject(),
		instrumentationConfig(reportsConfigName, instrumentedNodesNamespace),
		cronJobPod("reports-28901234-p2xzq", agentsNodeName),
		otherCronJobPod,
		sameNameOtherKind,
		podOnNode("no-owner", otherNodeName),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	_, err := r.Reconcile(ctx, instrumentationConfigRequest(reportsConfigName))

	require.NoError(t, err)
	requireNodeStamped(t, c, agentsNodeName)
	requireNodeNotStamped(t, c, otherNodeName)
}

func TestInstrumentationConfigReconciler_MissingWorkloadIsANoOp(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		nodeNamed(agentsNodeName),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, instrumentationConfigRequest(checkoutConfigName))

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	requireNodeNotStamped(t, c, agentsNodeName)
}

func TestInstrumentationConfigReconciler_InvalidNameIsAnError(t *testing.T) {
	ctx := instrumentedNodesContext()
	r := &InstrumentationConfigReconciler{Client: newInstrumentedNodesClient(t), NodeLabelRetention: time.Minute}

	tests := []struct {
		name        string
		requestName string
	}{
		{name: "no kind prefix", requestName: "checkout"},
		{name: "unsupported kind", requestName: "replicaset-checkout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Reconcile(ctx, instrumentationConfigRequest(tt.requestName))

			assert.Error(t, err)
		})
	}
}

func TestInstrumentationConfigReconciler_WorkloadGetErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
				if _, isDeployment := obj.(*appsv1.Deployment); isDeployment {
					return apierrors.NewInternalError(errors.New("deployment cache is cold"))
				}
				return nil
			},
		},
		checkoutDeployment(),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	_, err := r.Reconcile(ctx, instrumentationConfigRequest(checkoutConfigName))

	assert.ErrorContains(t, err, "deployment cache is cold")
}

func TestInstrumentationConfigReconciler_PodListErrorIsPropagated(t *testing.T) {
	ctx := instrumentedNodesContext()
	tests := []struct {
		name        string
		requestName string
		objects     []client.Object
	}{
		{
			name:        "listing the pods of a deployment",
			requestName: checkoutConfigName,
			objects:     []client.Object{checkoutDeployment()},
		},
		{
			name:        "listing the pods of a cron job",
			requestName: reportsConfigName,
			objects:     []client.Object{reportsCronJobObject()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newInstrumentedNodesClientWithInterceptor(t,
				interceptor.Funcs{
					List: func(_ context.Context, _ client.WithWatch, _ client.ObjectList, _ ...client.ListOption) error {
						return errors.New("pod cache is not synced")
					},
				},
				tt.objects...,
			)
			r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

			_, err := r.Reconcile(ctx, instrumentationConfigRequest(tt.requestName))

			assert.ErrorContains(t, err, "pod cache is not synced")
		})
	}
}

func TestInstrumentationConfigReconciler_ConflictIsRetriedWithoutAnError(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
				return apierrors.NewConflict(schema.GroupResource{Resource: "nodes"}, agentsNodeName, errors.New("the object has been modified"))
			},
		},
		nodeNamed(agentsNodeName),
		checkoutDeployment(),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		instrumentationConfig(checkoutConfigName, instrumentedNodesNamespace),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	result, err := r.Reconcile(ctx, instrumentationConfigRequest(checkoutConfigName))

	require.NoError(t, err)
	assert.True(t, result.Requeue)
}

// One node that cannot be stamped must not hide the failure, and must not stop
// the other nodes of the same workload from being stamped.
func TestInstrumentationConfigReconciler_AFailedNodeIsReportedAndTheOthersAreStillStamped(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClientWithInterceptor(t,
		interceptor.Funcs{
			Patch: func(ctx context.Context, inner client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				if obj.GetName() == otherNodeName {
					return apierrors.NewForbidden(schema.GroupResource{Resource: "nodes"}, otherNodeName, errors.New("no permission to label nodes"))
				}
				return inner.Patch(ctx, obj, patch, opts...)
			},
		},
		nodeNamed(agentsNodeName),
		nodeNamed(otherNodeName),
		checkoutDeployment(),
		instrumentationConfig(checkoutConfigName, instrumentedNodesNamespace),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		deploymentPod("checkout-7d4c8b5f9b-h9vlk", otherNodeName),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	_, err := r.Reconcile(ctx, instrumentationConfigRequest(checkoutConfigName))

	assert.ErrorContains(t, err, "no permission to label nodes")
	requireNodeStamped(t, c, agentsNodeName)
	requireNodeNotStamped(t, c, otherNodeName)
}

// A deleted instrumentation config (the source was removed) also reconciles: its
// workload keeps running, so the stamps stay until the retention runs out, and
// the reconcile has to come back at the earlier of the two deadlines.
func TestInstrumentationConfigReconciler_RequeuesForTheEarliestRetentionDeadline(t *testing.T) {
	ctx := instrumentedNodesContext()
	expiringSoon := nodeStampedAt(agentsNodeName, stampedAgo(9*time.Minute))
	expiringLater := nodeStampedAt(otherNodeName, stampedAgo(time.Minute))
	c := newInstrumentedNodesClient(t,
		expiringSoon,
		expiringLater,
		checkoutDeployment(),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		deploymentPod("checkout-7d4c8b5f9b-h9vlk", otherNodeName),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: 10 * time.Minute}

	result, err := r.Reconcile(ctx, instrumentationConfigRequest(checkoutConfigName))

	require.NoError(t, err)
	assert.Greater(t, result.RequeueAfter, time.Duration(0))
	assert.LessOrEqual(t, result.RequeueAfter, time.Minute,
		"the node whose retention expires first sets the deadline")
	assert.NotEmpty(t, requireNodeStamped(t, c, agentsNodeName))
	assert.NotEmpty(t, requireNodeStamped(t, c, otherNodeName))
}

// ****************
// nodeNamesForWorkload() tests
// ****************

func TestNodeNamesForWorkload_AWorkloadKindWithNoObjectHasNoNodes(t *testing.T) {
	ctx := instrumentedNodesContext()
	r := &InstrumentationConfigReconciler{Client: newInstrumentedNodesClient(t), NodeLabelRetention: time.Minute}

	nodes, err := r.nodeNamesForWorkload(ctx, k8sconsts.PodWorkload{
		Namespace: instrumentedNodesNamespace,
		Name:      checkoutWorkload,
		Kind:      k8sconsts.WorkloadKind("Unsupported"),
	})

	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestNodeNamesForWorkload_AWorkloadWithoutASelectorHasNoNodes(t *testing.T) {
	ctx := instrumentedNodesContext()
	selectorless := checkoutDeployment()
	selectorless.Spec.Selector = nil
	c := newInstrumentedNodesClient(t, selectorless, deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName))
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	nodes, err := r.nodeNamesForWorkload(ctx, k8sconsts.PodWorkload{
		Namespace: instrumentedNodesNamespace,
		Name:      checkoutWorkload,
		Kind:      k8sconsts.WorkloadKindDeployment,
	})

	require.NoError(t, err)
	assert.Empty(t, nodes, "without a selector there is no way to tell which pods belong to the workload")
}

func TestNodeNamesForWorkload_ReportsEachNodeOnce(t *testing.T) {
	ctx := instrumentedNodesContext()
	c := newInstrumentedNodesClient(t,
		checkoutDeployment(),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		deploymentPod("checkout-7d4c8b5f9b-h9vlk", agentsNodeName),
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	nodes, err := r.nodeNamesForWorkload(ctx, k8sconsts.PodWorkload{
		Namespace: instrumentedNodesNamespace,
		Name:      checkoutWorkload,
		Kind:      k8sconsts.WorkloadKindDeployment,
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]struct{}{agentsNodeName: {}}, nodes)
}

// The nodes of a workload are the nodes of the pods in its own namespace; a
// namespace-wide list would drag in every cluster namespace.
func TestNodeNamesForWorkload_PodsAreScopedToTheWorkloadNamespace(t *testing.T) {
	ctx := instrumentedNodesContext()
	sameSelectorOtherNamespace := deploymentPod("checkout-7d4c8b5f9b-elsewhere", otherNodeName)
	sameSelectorOtherNamespace.Namespace = instrumentedNodesOtherNamespace
	c := newInstrumentedNodesClient(t,
		checkoutDeployment(),
		deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName),
		sameSelectorOtherNamespace,
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	nodes, err := r.nodeNamesForWorkload(ctx, k8sconsts.PodWorkload{
		Namespace: instrumentedNodesNamespace,
		Name:      checkoutWorkload,
		Kind:      k8sconsts.WorkloadKindDeployment,
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]struct{}{agentsNodeName: {}}, nodes)
}

// Cron job pods are found by owner reference rather than by a selector, so both
// the namespace and the workload kind have to be checked by hand.
func TestNodeNamesForWorkload_CronJobPodsAreScopedToTheirNamespaceAndKind(t *testing.T) {
	ctx := instrumentedNodesContext()
	sameOwnerOtherNamespace := cronJobPod("reports-28901234-elsewhere", otherNodeName)
	sameOwnerOtherNamespace.Namespace = instrumentedNodesOtherNamespace
	// a deployment that happens to share the name of the cron job
	sameNameOtherKind := podOnNode("reports-7d4c8b5f9b-w4trp", otherNodeName)
	sameNameOtherKind.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "apps/v1",
		Kind:       "ReplicaSet",
		Name:       "reports-7d4c8b5f9b",
	}}
	c := newInstrumentedNodesClient(t,
		reportsCronJobObject(),
		cronJobPod("reports-28901234-p2xzq", agentsNodeName),
		sameOwnerOtherNamespace,
		sameNameOtherKind,
	)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	nodes, err := r.nodeNamesForWorkload(ctx, k8sconsts.PodWorkload{
		Namespace: instrumentedNodesNamespace,
		Name:      reportsCronJob,
		Kind:      k8sconsts.WorkloadKindCronJob,
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]struct{}{agentsNodeName: {}}, nodes)
}

// An unscheduled pod has no node, and the empty string is not one: it would be
// carried into a sync for a node that does not exist.
func TestNodeNamesForWorkload_UnscheduledPodsAreSkipped(t *testing.T) {
	ctx := instrumentedNodesContext()

	tests := []struct {
		name     string
		objects  []client.Object
		workload k8sconsts.PodWorkload
	}{
		{
			name:    "a pod of a deployment",
			objects: []client.Object{checkoutDeployment(), deploymentPod("checkout-7d4c8b5f9b-pending", "")},
			workload: k8sconsts.PodWorkload{
				Namespace: instrumentedNodesNamespace,
				Name:      checkoutWorkload,
				Kind:      k8sconsts.WorkloadKindDeployment,
			},
		},
		{
			name:    "a pod of a cron job",
			objects: []client.Object{reportsCronJobObject(), cronJobPod("reports-28901234-pending", "")},
			workload: k8sconsts.PodWorkload{
				Namespace: instrumentedNodesNamespace,
				Name:      reportsCronJob,
				Kind:      k8sconsts.WorkloadKindCronJob,
			},
		},
		{
			name:    "a static pod",
			objects: []client.Object{staticPod("kube-apiserver", "")},
			workload: k8sconsts.PodWorkload{
				Namespace: instrumentedNodesNamespace,
				Name:      "kube-apiserver",
				Kind:      k8sconsts.WorkloadKindStaticPod,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &InstrumentationConfigReconciler{
				Client:             newInstrumentedNodesClient(t, tt.objects...),
				NodeLabelRetention: time.Minute,
			}

			nodes, err := r.nodeNamesForWorkload(ctx, tt.workload)

			require.NoError(t, err)
			assert.Empty(t, nodes)
		})
	}
}

func TestNodeNamesForWorkload_UsesTheWorkloadOwnSelector(t *testing.T) {
	ctx := instrumentedNodesContext()
	// the deployment selects on a label the pods of the "checkout" fixture do not
	// carry, so name-based matching would wrongly claim the pod
	pickySelector := checkoutDeployment()
	pickySelector.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{"app": "checkout", "tier": "web"}}
	c := newInstrumentedNodesClient(t, pickySelector, deploymentPod("checkout-7d4c8b5f9b-p2xzq", agentsNodeName))
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	nodes, err := r.nodeNamesForWorkload(ctx, k8sconsts.PodWorkload{
		Namespace: instrumentedNodesNamespace,
		Name:      checkoutWorkload,
		Kind:      k8sconsts.WorkloadKindDeployment,
	})

	require.NoError(t, err)
	assert.Empty(t, nodes)
}

func TestNodeNamesForWorkload_DaemonSetPodsAreMatchedByTheirSelector(t *testing.T) {
	ctx := instrumentedNodesContext()
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "log-shipper", Namespace: instrumentedNodesNamespace},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "log-shipper"}},
		},
	}
	shipperPod := podOnNode("log-shipper-x9jkq", agentsNodeName)
	shipperPod.Labels = map[string]string{"app": "log-shipper"}
	c := newInstrumentedNodesClient(t, daemonSet, shipperPod)
	r := &InstrumentationConfigReconciler{Client: c, NodeLabelRetention: time.Minute}

	nodes, err := r.nodeNamesForWorkload(ctx, k8sconsts.PodWorkload{
		Namespace: instrumentedNodesNamespace,
		Name:      "log-shipper",
		Kind:      k8sconsts.WorkloadKindDaemonSet,
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]struct{}{agentsNodeName: {}}, nodes)
}
