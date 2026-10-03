package graph

import (
	"strings"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	generatedstatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// wrEveryWorkloadField selects every K8sWorkload field that has a resolver of its own.
// It is shared by the two query resolvers so that "the two code paths agree" is checked
// over the whole surface rather than the fields a test happened to list.
const wrEveryWorkloadField = `
	id { namespace kind name }
	serviceName
	numberOfInstances
	dataStreamNames
	rollbackOccurred
	markedForInstrumentation { markedForInstrumentation decisionEnum message }
	runtimeInfo {
		completed
		completedStatus { name status reasonEnum }
		detectedLanguages
		containers { containerName language runtimeVersion }
	}
	agentEnabled {
		agentEnabled
		enabledStatus { name status reasonEnum }
		containers { containerName agentEnabled otelDistroName }
	}
	autoRollback { autoRollbackStatus { name status reasonEnum } rollbackOccurred }
	containers { containerName agentEnabled { agentEnabled } runtimeInfo { language } }
	conditions {
		runtimeDetection { name status reasonEnum }
		agentInjectionEnabled { name status reasonEnum }
		rollout { name status reasonEnum }
		podsManifestInjection { name status reasonEnum }
		autoRollback { name status reasonEnum }
		agentInjected { name status reasonEnum }
		processesAgentHealth { name status reasonEnum }
		expectingTelemetry { name status reasonEnum }
	}
	workloadOdigosHealthStatus { name status reasonEnum message }
	podsAgentInjectionStatus { name status reasonEnum }
	podsHealthStatus { name status reasonEnum }
	podsOdigosHealthStatus { name status reasonEnum }
	workloadHealthStatus { name status reasonEnum }
	processesHealthStatus { name status reasonEnum }
	pods { podName containers { containerName } }
	telemetryMetrics { totalDataSentBytes throughputBytes }`

// wrSteadyStateObjects is a workload the whole control loop has converged on: reconciled
// InstrumentationConfig, an enabled Source with a data stream, one pod carrying the current
// agent hash, and a healthy InstrumentationInstance for its container.
func wrSteadyStateObjects() []client.Object {
	podName := wrPodName(wrWorkloadName, 0)
	return []client.Object{
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 3),
		wrReconciledIC(),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrPod(wrPodFixture{
			name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
		}),
		wrHealthyInstance(podName, wrContainerName, "9001"),
	}
}

// TestTheTwoWorkloadQueriesAgreeOnEveryFieldTheyBothResolve is the highest-reach test in
// this file. workloads(filter:) pre-computes 14 fields in populateWorkloadFields so that
// gqlgen's per-field resolvers short-circuit on their `if obj.X != nil` guards, while
// workloadsByIds — the query the UI's live refresh uses — returns bare ids and lets every
// lazy resolver run. Two full implementations of the same answer therefore exist side by
// side, and nothing but this test makes them agree: the list view and the detail view can
// describe the same workload differently with no compiler or schema error.
func TestTheTwoWorkloadQueriesAgreeOnEveryFieldTheyBothResolve(t *testing.T) {
	objects := wrSteadyStateObjects()

	eagerCtx, eagerResolver := wrHarness(t, objects...)
	eager := wrOnlyWorkload(t, wrExecute(t, eagerCtx, eagerResolver,
		`query($filter: WorkloadFilter) { workloads(filter: $filter) { `+wrEveryWorkloadField+` } }`,
		wrSingleWorkloadVars()))

	lazyCtx, lazyResolver, _ := wrUnloadedHarness(t, wrInterceptors{}, objects...)
	lazyData := wrExecute(t, lazyCtx, lazyResolver,
		`query($ids: [K8sWorkloadIdInput!]!) { workloadsByIds(ids: $ids) { `+wrEveryWorkloadField+` } }`,
		map[string]interface{}{"ids": []interface{}{map[string]interface{}{
			"namespace": wrAppNamespace,
			"kind":      string(model.K8sResourceKindDeployment),
			"name":      wrWorkloadName,
		}}})
	lazyList := wrList(t, lazyData, "workloadsByIds")
	require.Len(t, lazyList, 1)
	lazy := wrObject(t, lazyList[0])

	// conditions is compared field by field below, because the two paths are known to
	// disagree on exactly one of its members.
	agreeing := []string{
		"id", "serviceName", "numberOfInstances", "dataStreamNames", "rollbackOccurred",
		"markedForInstrumentation", "runtimeInfo", "agentEnabled", "autoRollback",
		"containers", "workloadOdigosHealthStatus", "podsAgentInjectionStatus",
		"podsHealthStatus", "podsOdigosHealthStatus", "workloadHealthStatus",
		"processesHealthStatus", "pods", "telemetryMetrics",
	}
	for _, field := range agreeing {
		require.Contains(t, eager, field, "the eager query did not return %q", field)
		assert.Equal(t, eager[field], lazy[field],
			"workloads and workloadsByIds disagree about %q", field)
	}

	eagerConditions := wrChild(t, eager, "conditions")
	lazyConditions := wrChild(t, lazy, "conditions")
	for _, condition := range []string{
		"runtimeDetection", "agentInjectionEnabled", "rollout", "podsManifestInjection",
		"agentInjected", "processesAgentHealth", "expectingTelemetry",
	} {
		assert.Equal(t, eagerConditions[condition], lazyConditions[condition],
			"the two paths disagree about conditions.%s", condition)
	}

	// Characterised, not asserted as desirable: populateWorkloadFields never assigns
	// AutoRollback into K8sWorkloadConditions, so the list view reports a null
	// conditions.autoRollback for a workload whose detail view reports one. The
	// top-level autoRollback field, compared above, is populated on both paths.
	assert.Nil(t, eagerConditions["autoRollback"],
		"the eager path is known not to compute conditions.autoRollback")
	assert.NotNil(t, lazyConditions["autoRollback"],
		"the lazy path does compute conditions.autoRollback")

	// anti-vacuity: the comparison above is worthless if the fields are all null.
	assert.Equal(t, wrServiceName, eager["serviceName"])
	assert.EqualValues(t, 3, eager["numberOfInstances"])
	assert.Len(t, wrList(t, eager, "pods"), 1)
	assert.Len(t, wrList(t, eager, "containers"), 1)
	require.Len(t, agreeing, 18)
}

// TestWorkloadsByIdsRefusesIdsInIgnoredNamespaces covers the filter in SetWorkloadIdsDirect.
// Without it the query would hand back data for any uninstrumented workload in a namespace
// the operator configured odigos to ignore, which is the resource the setting exists to
// hide. The complementary half proves an instrumented workload in the same ignored
// namespace is still returned, because it was explicitly opted in.
func TestWorkloadsByIdsRefusesIdsInIgnoredNamespaces(t *testing.T) {
	requestBoth := func(t *testing.T, extra ...client.Object) []interface{} {
		objects := append([]client.Object{
			wrEffectiveConfigMap("configVersion: 1\nignoredNamespaces:\n  - " + wrOtherNamespace + "\n"),
			wrNamespaceObject(wrAppNamespace),
			wrNamespaceObject(wrOtherNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			wrDeployment(wrOtherNamespace, wrOtherName, 1),
		}, extra...)

		ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{}, objects...)
		data := wrExecute(t, ctx, r,
			`query($ids: [K8sWorkloadIdInput!]!) { workloadsByIds(ids: $ids) { id { namespace name } } }`,
			map[string]interface{}{"ids": []interface{}{
				map[string]interface{}{"namespace": wrAppNamespace, "kind": "Deployment", "name": wrWorkloadName},
				map[string]interface{}{"namespace": wrOtherNamespace, "kind": "Deployment", "name": wrOtherName},
			}})
		return wrList(t, data, "workloadsByIds")
	}

	t.Run("an uninstrumented workload in an ignored namespace is dropped", func(t *testing.T) {
		got := requestBoth(t)
		require.Len(t, got, 1)
		assert.Equal(t, wrAppNamespace, wrString(t, wrChild(t, wrObject(t, got[0]), "id"), "namespace"))
	})

	t.Run("an instrumented workload in an ignored namespace is still returned", func(t *testing.T) {
		got := requestBoth(t, wrIC(wrOtherNamespace, wrOtherName))
		require.Len(t, got, 2)
		namespaces := []string{}
		for _, raw := range got {
			namespaces = append(namespaces, wrString(t, wrChild(t, wrObject(t, raw), "id"), "namespace"))
		}
		assert.ElementsMatch(t, []string{wrAppNamespace, wrOtherNamespace}, namespaces)
	})
}

// TestServiceNameFallsBackToTheSourceWhenTheInstrumentationConfigIsGone covers the
// three-way resolution. Disabling a Source deletes its InstrumentationConfig, so without
// the fallback the configured otel service name disappears from the UI the moment a user
// disables the source — and reappears if they re-enable it, which reads as data loss.
func TestServiceNameFallsBackToTheSourceWhenTheInstrumentationConfigIsGone(t *testing.T) {
	withServiceNameOnSource := func(s *v1alpha1.Source) { s.Spec.OtelServiceName = "name-from-the-source" }

	cases := map[string]struct {
		objects []client.Object
		want    *string
	}{
		"the instrumentation config wins when it has a service name": {
			objects: []client.Object{
				wrMainIC(),
				wrSourceWith(false, withServiceNameOnSource),
			},
			want: wrStr(wrServiceName),
		},
		"the source is used when the instrumentation config has no service name": {
			objects: []client.Object{
				wrMainIC(func(ic *v1alpha1.InstrumentationConfig) { ic.Spec.ServiceName = "" }),
				wrSourceWith(false, withServiceNameOnSource),
			},
			want: wrStr("name-from-the-source"),
		},
		"the source is used when there is no instrumentation config at all": {
			objects: []client.Object{wrSourceWith(true, withServiceNameOnSource)},
			want:    wrStr("name-from-the-source"),
		},
		"nothing is reported when neither carries a service name": {
			objects: []client.Object{wrSourceWith(false)},
			want:    nil,
		},
		"nothing is reported when the workload has no source and no config": {
			objects: nil,
			want:    nil,
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, r := wrHarness(t, append([]client.Object{
				wrNamespaceObject(wrAppNamespace),
				wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			}, tt.objects...)...)
			id := wrWorkloadID()
			got, err := r.K8sWorkload().ServiceName(ctx, &model.K8sWorkload{ID: &id})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func wrSourceWith(disabled bool, mutators ...func(*v1alpha1.Source)) *v1alpha1.Source {
	s := wrSource(wrAppNamespace, wrWorkloadName, disabled, wrDataStream)
	for _, m := range mutators {
		m(s)
	}
	return s
}

// TestServiceNameShortCircuitsAPrePopulatedValue pins that the eager list path is not
// re-resolved. Driven against a cluster whose answer differs, so a missing guard is visible
// rather than merely unobserved.
func TestServiceNameShortCircuitsAPrePopulatedValue(t *testing.T) {
	ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1), wrMainIC())
	id := wrWorkloadID()

	precomputed := wrStr("already-known")
	got, err := r.K8sWorkload().ServiceName(ctx, &model.K8sWorkload{ID: &id, ServiceName: precomputed})
	require.NoError(t, err)
	assert.Same(t, precomputed, got)

	fresh, err := r.K8sWorkload().ServiceName(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, fresh)
	assert.Equal(t, wrServiceName, *fresh, "without the guard the cluster value is used")
}

// TestMarkedForInstrumentationIsTriStateAndOnlyFalseWhenExplicitlyDisabled pins the
// distinction between "the user turned this off" and "nobody has looked at this workload
// yet". Both render as not-instrumented, but only the first is a decision: collapsing the
// nil into false would make every workload in the cluster look deliberately excluded.
func TestMarkedForInstrumentationIsTriStateAndOnlyFalseWhenExplicitlyDisabled(t *testing.T) {
	cases := map[string]struct {
		objects    []client.Object
		wantMarked *bool
		wantReason v1alpha1.MarkedForInstrumentationReason
	}{
		"an enabled workload source": {
			objects:    []client.Object{wrSource(wrAppNamespace, wrWorkloadName, false)},
			wantMarked: wrBool(true),
			wantReason: v1alpha1.MarkedForInstrumentationReasonWorkloadSource,
		},
		"an explicitly disabled workload source": {
			objects:    []client.Object{wrSource(wrAppNamespace, wrWorkloadName, true)},
			wantMarked: wrBool(false),
			wantReason: v1alpha1.MarkedForInstrumentationReasonWorkloadSourceDisabled,
		},
		"no source at all": {
			objects:    nil,
			wantMarked: nil,
			wantReason: v1alpha1.MarkedForInstrumentationReasonNoSource,
		},
		"an enabled namespace source and no workload source": {
			objects:    []client.Object{wrNamespaceSource(wrAppNamespace, false)},
			wantMarked: wrBool(true),
			wantReason: v1alpha1.MarkedForInstrumentationReasonNamespaceSource,
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, r := wrHarness(t, append([]client.Object{
				wrNamespaceObject(wrAppNamespace),
				wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			}, tt.objects...)...)
			id := wrWorkloadID()
			got, err := r.K8sWorkload().MarkedForInstrumentation(ctx, &model.K8sWorkload{ID: &id})
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantMarked, got.MarkedForInstrumentation)
			assert.Equal(t, string(tt.wantReason), got.DecisionEnum)
			assert.NotEmpty(t, got.Message, "the decision must carry an explanation")
		})
	}
}

// TestWorkloadOdigosHealthStatusGatesStaticPodsOnTheOnPremTier covers both halves of a
// licensing gate. A gate that always fires hides the real health of every static pod on
// on-prem; one that never fires leaves community users staring at a permanently unhealthy
// workload they cannot fix.
func TestWorkloadOdigosHealthStatusGatesStaticPodsOnTheOnPremTier(t *testing.T) {
	cases := map[string]struct {
		tier         model.Tier
		kind         model.K8sResourceKind
		wantOverride bool
	}{
		"a static pod on the community tier is reported as an enterprise feature": {
			tier: model.TierCommunity, kind: model.K8sResourceKindStaticPod, wantOverride: true,
		},
		"a static pod on the cloud tier is reported as an enterprise feature": {
			tier: model.TierCloud, kind: model.K8sResourceKindStaticPod, wantOverride: true,
		},
		"a static pod on the on-prem tier is evaluated normally": {
			tier: model.TierOnprem, kind: model.K8sResourceKindStaticPod, wantOverride: false,
		},
		"a deployment on the community tier is evaluated normally": {
			tier: model.TierCommunity, kind: model.K8sResourceKindDeployment, wantOverride: false,
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, r := wrHarness(t, wrSteadyStateObjects()...)
			wrSetTier(t, tt.tier)

			id := wrWorkloadID()
			id.Kind = tt.kind
			got, err := r.K8sWorkload().WorkloadOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
			require.NoError(t, err)
			require.NotNil(t, got)

			if tt.wantOverride {
				assert.Equal(t, model.DesiredStateProgressUnsupported, got.Status)
				assert.Equal(t, wrStr(string(status.WorkloadOdigosHealthStatusReasonEnterpriseFeature)), got.ReasonEnum)
				return
			}
			assert.NotEqual(t, wrStr(string(status.WorkloadOdigosHealthStatusReasonEnterpriseFeature)), got.ReasonEnum,
				"the enterprise gate must not fire here")
		})
	}
}

// wrStaticPod builds the pod shape workload.IsStaticPod accepts: owned by a Node, sourced
// from a file by the kubelet, and carrying the virtual-static-pod label that the synthetic
// selector fetchWorkloadManifests builds matches on.
func wrStaticPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: wrAppNamespace,
			Labels: map[string]string{
				k8sconsts.OdigosVirtualStaticPodNameLabel: name,
				k8sconsts.OdigosAgentsMetaHashLabel:       wrAgentsMetaHash,
			},
			Annotations:       map[string]string{"kubernetes.io/config.source": "file"},
			CreationTimestamp: metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)),
			OwnerReferences:   []metav1.OwnerReference{{APIVersion: "v1", Kind: "Node", Name: "wr-node", UID: "node-uid"}},
		},
		Spec: corev1.PodSpec{
			NodeName:   "wr-node",
			Containers: []corev1.Container{{Name: wrContainerName}},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: wrContainerName, Ready: true, Started: wrBool(true),
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			}},
		},
	}
}

// TestTheWorkloadsListReadsTheTierFromTheClusterForItsStaticPodGate covers the tier read
// in queryResolver.Workloads, which is resolved once per request and threaded into
// populateWorkloadFields. It is the list-view half of the gate that
// TestWorkloadOdigosHealthStatusGatesStaticPodsOnTheOnPremTier covers for the detail page;
// without it, hard-coding the tier is invisible from the list view.
func TestTheWorkloadsListReadsTheTierFromTheClusterForItsStaticPodGate(t *testing.T) {
	staticPodName := "wr-static-pod"
	objects := []client.Object{
		wrNamespaceObject(wrAppNamespace),
		wrStaticPod(staticPodName),
		wrIC(wrAppNamespace, staticPodName),
	}
	filter := &model.WorkloadFilter{
		Namespace: wrStr(wrAppNamespace),
		Kind:      wrKind(model.K8sResourceKindStaticPod),
		Name:      wrStr(staticPodName),
	}

	for tier, wantEnterpriseGate := range map[model.Tier]bool{
		model.TierCommunity: true,
		model.TierOnprem:    false,
	} {
		t.Run(string(tier), func(t *testing.T) {
			ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{}, objects...)
			wrSetTier(t, tier)

			got, err := r.Query().Workloads(ctx, filter)
			require.NoError(t, err)
			require.Len(t, got, 1)
			require.NotNil(t, got[0].ID)
			assert.Equal(t, model.K8sResourceKindStaticPod, got[0].ID.Kind)
			require.NotNil(t, got[0].WorkloadOdigosHealthStatus)

			gated := got[0].WorkloadOdigosHealthStatus.ReasonEnum != nil &&
				*got[0].WorkloadOdigosHealthStatus.ReasonEnum ==
					string(status.WorkloadOdigosHealthStatusReasonEnterpriseFeature)
			assert.Equal(t, wantEnterpriseGate, gated,
				"static pod gating on tier %q", tier)
		})
	}
}

// TestWorkloadOdigosHealthStatusReportsAWorkloadWithNoInstrumentationConfigAsDisabled
// covers the else branch that synthesises a condition when there is no IC to derive one
// from. The condition has to exist: workloadOdigosHealthStatus is the badge every row of
// the workload list renders.
func TestWorkloadOdigosHealthStatusReportsAWorkloadWithNoInstrumentationConfigAsDisabled(t *testing.T) {
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			containers: []wrContainerFixture{wrHealthyContainer(wrContainerName, "")},
		}),
	)

	id := wrWorkloadID()
	got, err := r.K8sWorkload().WorkloadOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, status.WorkloadOdigosHealthStatus, got.Name)
	assert.Equal(t, model.DesiredStateProgressDisabled, got.Status)
	assert.Equal(t, wrStr(string(status.WorkloadOdigosHealthStatusReasonDisabled)), got.ReasonEnum)
	assert.Equal(t, "workload is not marked for instrumentation", got.Message)
}

// wrUninjectedPodObjects is a reconciled, agent-enabled workload whose single running pod
// never received the agent: the instrumentor's pod webhook did not stamp the agents-meta-hash
// label on it. This is the state an operator sees after enabling a source without a rollout.
func wrUninjectedPodObjects() []client.Object {
	return []client.Object{
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		// a distro that never reports InstrumentationInstances keeps the processes-health
		// condition out of Waiting without needing a live metrics consumer.
		wrReconciledIC(func(ic *v1alpha1.InstrumentationConfig) {
			ic.Spec.Containers[0].OtelDistroName = wrSilentDistro
		}),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			containers: []wrContainerFixture{wrHealthyContainer(wrContainerName, wrSilentDistro)},
		}),
	}
}

// TestWorkloadOdigosHealthStatusRewritesAnAllSuccessAggregateAsCollectingTelemetry covers
// the final rewrite, which is the only thing that turns a green aggregate into the badge
// and message the UI actually shows.
func TestWorkloadOdigosHealthStatusRewritesAnAllSuccessAggregateAsCollectingTelemetry(t *testing.T) {
	ctx, r := wrHarness(t, wrUninjectedPodObjects()...)

	id := wrWorkloadID()
	got, err := r.K8sWorkload().WorkloadOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, status.WorkloadOdigosHealthStatus, got.Name)
	assert.Equal(t, model.DesiredStateProgressSuccess, got.Status)
	assert.Equal(t, wrStr(string(status.WorkloadOdigosHealthStatusReasonCollectingTelemetry)), got.ReasonEnum)
	assert.Equal(t, "source is instrumented and telemetry is being collected", got.Message)
}

// TestWorkloadOdigosHealthStatusOmitsTheAgentInjectedConditionThatTheListViewIncludes
// characterises a divergence between the two implementations of the same badge rather than
// asserting a desired behaviour.
//
// populateWorkloadFields, used by workloads(filter:), feeds CalculateAgentInjectedStatus
// into the aggregation. The lazy resolver here does not — it appends runtime inspection,
// agent-injection-enabled, pods-manifest-injection, processes health and expecting-telemetry,
// and nothing else. So for a workload whose running pod never got the agent, the list view
// shows a Notice telling the operator to roll the workload out while the detail view shows
// a green "telemetry is being collected", for the same workload at the same moment.
func TestWorkloadOdigosHealthStatusOmitsTheAgentInjectedConditionThatTheListViewIncludes(t *testing.T) {
	objects := wrUninjectedPodObjects()
	id := wrWorkloadID()

	lazyCtx, lazyResolver := wrHarness(t, objects...)
	lazy, err := lazyResolver.K8sWorkload().WorkloadOdigosHealthStatus(lazyCtx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, lazy)

	eagerCtx, eagerResolver := wrHarness(t, objects...)
	eager := wrOnlyWorkload(t, wrExecute(t, eagerCtx, eagerResolver,
		`query($filter: WorkloadFilter) {
			workloads(filter: $filter) {
				workloadOdigosHealthStatus { status reasonEnum }
				conditions { agentInjected { status reasonEnum } }
			}
		}`, wrSingleWorkloadVars()))
	eagerHealth := wrChild(t, eager, "workloadOdigosHealthStatus")

	// the condition both paths compute, and which is the worst one present.
	agentInjected := wrChild(t, wrChild(t, eager, "conditions"), "agentInjected")
	assert.Equal(t, string(model.DesiredStateProgressNotice), agentInjected["status"])
	assert.Equal(t, string(status.AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded),
		agentInjected["reasonEnum"])

	// the list view surfaces it; the detail page reports success.
	assert.Equal(t, string(model.DesiredStateProgressNotice), eagerHealth["status"])
	assert.Equal(t, model.DesiredStateProgressSuccess, lazy.Status)
	assert.NotEqual(t, eagerHealth["status"], string(lazy.Status),
		"the two implementations of workloadOdigosHealthStatus must eventually agree")
}

// TestWorkloadOdigosHealthStatusSurfacesTheWorstUnderlyingCondition is the anti-drop
// invariant for the aggregation: each row makes exactly one of the inputs the uniquely
// worst one, so dropping any single condition from the append becomes observable. A fixture
// where every condition is Success cannot see any of them being lost.
func TestWorkloadOdigosHealthStatusSurfacesTheWorstUnderlyingCondition(t *testing.T) {
	podName := wrPodName(wrWorkloadName, 0)
	silentContainer := wrHealthyContainer(wrContainerName, wrSilentDistro)
	silentDistro := func(ic *v1alpha1.InstrumentationConfig) {
		ic.Spec.Containers[0].OtelDistroName = wrSilentDistro
	}

	cases := map[string]struct {
		objects []client.Object
		// wantStatus is optional: the precedence assertions below only need the reason.
		wantStatus model.DesiredStateProgress
		wantReason string
	}{
		"runtime detection failed": {
			objects: []client.Object{
				wrReconciledIC(silentDistro, wrWithConditionReason(
					v1alpha1.RuntimeDetectionStatusConditionType,
					string(v1alpha1.RuntimeDetectionReasonError))),
				wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
					agentsMetaHash: wrAgentsMetaHash, containers: []wrContainerFixture{silentContainer}}),
			},
			wantStatus: model.DesiredStateProgressFailure,
			wantReason: string(v1alpha1.RuntimeDetectionReasonError),
		},
		"another agent was detected so odigos did not enable its own": {
			objects: []client.Object{
				wrReconciledIC(silentDistro, wrWithConditionReason(
					generatedstatus.AgentEnabledType,
					string(generatedstatus.AgentEnabledReasonOtherAgentDetected))),
				wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
					agentsMetaHash: wrAgentsMetaHash, containers: []wrContainerFixture{silentContainer}}),
			},
			wantStatus: model.DesiredStateProgressNotice,
			// the generated calculators map the reason enum to its display title.
			wantReason: generatedstatus.AgentEnabledOtherAgentDetected.Title,
		},
		"a restart is required but automatic rollout is disabled": {
			objects: []client.Object{
				wrReconciledIC(silentDistro, wrWithConditionReason(
					generatedstatus.PodsManifestInjectionType,
					string(generatedstatus.PodsManifestInjectionReasonRestartRequiredAutoRolloutDisabled_Enabled))),
				wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
					agentsMetaHash: wrAgentsMetaHash, containers: []wrContainerFixture{silentContainer}}),
			},
			wantStatus: model.DesiredStateProgressNotice,
			wantReason: generatedstatus.PodsManifestInjectionRestartRequiredAutoRolloutDisabled_Enabled.Title,
		},
		"the workload has no running pods": {
			objects: []client.Object{
				wrReconciledIC(silentDistro),
			},
			wantStatus: model.DesiredStateProgressPending,
			wantReason: string(status.ExpectingTelemetryReasonNoRunningPod),
		},
		"an instrumented process reports an unhealthy agent": {
			objects: []client.Object{
				wrReconciledIC(),
				wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
					agentsMetaHash: wrAgentsMetaHash,
					containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)}}),
				wrInstance(wrInstanceFixture{
					name: "unhealthy", namespace: wrAppNamespace, workloadName: wrWorkloadName,
					podName: podName, containerName: wrContainerName, healthy: wrBool(false),
				}),
			},
			wantStatus: model.DesiredStateProgressFailure,
		},
	}

	distinct := map[string]string{}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, r := wrHarness(t, append([]client.Object{
				wrNamespaceObject(wrAppNamespace),
				wrDeployment(wrAppNamespace, wrWorkloadName, 1),
				wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
			}, tt.objects...)...)

			id := wrWorkloadID()
			got, err := r.K8sWorkload().WorkloadOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
			require.NoError(t, err)
			require.NotNil(t, got)
			if tt.wantStatus != "" {
				assert.Equal(t, tt.wantStatus, got.Status)
			}
			if tt.wantReason != "" {
				assert.Equal(t, wrStr(tt.wantReason), got.ReasonEnum)
			}
			require.NotNil(t, got.ReasonEnum)
			distinct[name] = got.Status.String() + "|" + *got.ReasonEnum
		})
	}

	seen := map[string]string{}
	for name, fingerprint := range distinct {
		if other, collision := seen[fingerprint]; collision {
			t.Fatalf("%q and %q are indistinguishable to the UI: %s", other, name, fingerprint)
		}
		seen[fingerprint] = name
	}
	require.Len(t, seen, len(cases))
}

// wrWithConditionReason replaces the reason of one already-present status condition,
// expressing "everything reconciled except this one thing".
func wrWithConditionReason(conditionType, reason string) func(*v1alpha1.InstrumentationConfig) {
	return func(ic *v1alpha1.InstrumentationConfig) {
		for i := range ic.Status.Conditions {
			if ic.Status.Conditions[i].Type == conditionType {
				ic.Status.Conditions[i].Reason = reason
				return
			}
		}
		panic("no condition of type " + conditionType + " in the fixture")
	}
}

// TestConditionsPopulatesEveryMemberOfTheConditionsObject is the completeness gate for the
// eight-field struct literal. A field left unassigned renders as null and the matching
// row of the workload detail page silently disappears.
func TestConditionsPopulatesEveryMemberOfTheConditionsObject(t *testing.T) {
	ctx, r := wrHarness(t, wrSteadyStateObjects()...)
	id := wrWorkloadID()

	got, err := r.K8sWorkload().Conditions(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, got)

	byField := map[string]*model.DesiredConditionStatus{
		"runtimeDetection":      got.RuntimeDetection,
		"agentInjectionEnabled": got.AgentInjectionEnabled,
		"rollout":               got.Rollout,
		"podsManifestInjection": got.PodsManifestInjection,
		"autoRollback":          got.AutoRollback,
		"agentInjected":         got.AgentInjected,
		"processesAgentHealth":  got.ProcessesAgentHealth,
		"expectingTelemetry":    got.ExpectingTelemetry,
	}
	names := map[string]string{}
	for field, condition := range byField {
		require.NotNil(t, condition, "conditions.%s was not populated", field)
		assert.NotEmpty(t, condition.Name, "conditions.%s has no name", field)
		names[field] = condition.Name
	}

	// every member must carry its own condition name, or two fields are wired to the
	// same calculator.
	seen := map[string]string{}
	for field, name := range names {
		if other, collision := seen[name]; collision {
			t.Fatalf("conditions.%s and conditions.%s both report the condition name %q", other, field, name)
		}
		seen[name] = field
	}
	require.Len(t, seen, 8)
}

// TestConditionsShortCircuitsAPrePopulatedValue completes the guard pair.
func TestConditionsShortCircuitsAPrePopulatedValue(t *testing.T) {
	ctx, r := wrHarness(t, wrSteadyStateObjects()...)
	id := wrWorkloadID()

	precomputed := &model.K8sWorkloadConditions{}
	got, err := r.K8sWorkload().Conditions(ctx, &model.K8sWorkload{ID: &id, Conditions: precomputed})
	require.NoError(t, err)
	assert.Same(t, precomputed, got)
}

// TestRuntimeInfoReportsDetectionAsIncompleteUntilAContainerHasRuntimeDetails covers the
// completed flag and the detected-languages list it gates. Reporting completed=true with
// no details makes the UI claim detection finished and found nothing.
func TestRuntimeInfoReportsDetectionAsIncompleteUntilAContainerHasRuntimeDetails(t *testing.T) {
	id := wrWorkloadID()

	t.Run("detection has not run yet", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1), wrMainIC())
		got, err := r.K8sWorkload().RuntimeInfo(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.False(t, got.Completed)
		assert.Nil(t, got.DetectedLanguages, "languages must stay null while detection is incomplete")
		assert.Empty(t, got.Containers)
		require.NotNil(t, got.CompletedStatus)
	})

	t.Run("detection completed for two containers", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			// seeded out of alphabetical order to exercise the sort.
			wrReconciledIC(func(ic *v1alpha1.InstrumentationConfig) {
				ic.Status.RuntimeDetailsByContainer = []v1alpha1.RuntimeDetailsByContainer{
					{ContainerName: wrSidecarName, Language: common.JavaProgrammingLanguage, RuntimeVersion: "21.0.1"},
					{ContainerName: wrContainerName, Language: common.GoProgrammingLanguage, RuntimeVersion: "1.26.0"},
				}
			}))
		got, err := r.K8sWorkload().RuntimeInfo(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.True(t, got.Completed)
		require.Len(t, got.Containers, 2)
		assert.Equal(t, []string{wrContainerName, wrSidecarName},
			[]string{got.Containers[0].ContainerName, got.Containers[1].ContainerName},
			"containers must be sorted by name")
		assert.Equal(t, wrStr("1.26.0"), got.Containers[0].RuntimeVersion)
		assert.ElementsMatch(t,
			[]model.ProgrammingLanguage{
				model.ProgrammingLanguage(common.GoProgrammingLanguage),
				model.ProgrammingLanguage(common.JavaProgrammingLanguage),
			},
			got.DetectedLanguages,
			"see TestTheDetectedLanguagesAreNotMembersOfTheSchemaEnum for the casing")
	})

	t.Run("there is no instrumentation config", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1))
		got, err := r.K8sWorkload().RuntimeInfo(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

// TestTheDetectedLanguagesAreNotMembersOfTheSchemaEnum characterises a bug rather than
// asserting a desired behaviour.
//
// The schema declares `enum ProgrammingLanguage { Unspecified Java Go JavaScript Python
// DotNet }`, but runtimeDetailsContainersToModel and collectEffectiveDetectedLanguages
// cast common.ProgrammingLanguage straight to model.ProgrammingLanguage, and those values
// are lower-case ("go", "java", ...). gqlgen's generated enum marshaler only quotes the
// string, so the API happily serves values that are not members of the enum it advertises,
// and a client generated from the schema gets a value outside its own union type.
// Several detected languages have no schema member at all, whatever the casing.
func TestTheDetectedLanguagesAreNotMembersOfTheSchemaEnum(t *testing.T) {
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrReconciledIC(func(ic *v1alpha1.InstrumentationConfig) {
			ic.Status.RuntimeDetailsByContainer = []v1alpha1.RuntimeDetailsByContainer{
				{ContainerName: wrContainerName, Language: common.GoProgrammingLanguage},
			}
		}))

	// what the API actually puts on the wire.
	got := wrOnlyWorkload(t, wrExecute(t, ctx, r,
		`query($filter: WorkloadFilter) {
			workloads(filter: $filter) {
				runtimeInfo { detectedLanguages containers { language } }
			}
		}`, wrSingleWorkloadVars()))
	runtimeInfo := wrChild(t, got, "runtimeInfo")
	assert.Equal(t, []interface{}{string(common.GoProgrammingLanguage)}, runtimeInfo["detectedLanguages"])

	containers := wrList(t, runtimeInfo, "containers")
	require.Len(t, containers, 1)
	served := wrString(t, wrObject(t, containers[0]), "language")
	assert.Equal(t, string(common.GoProgrammingLanguage), served)

	// and why that is wrong.
	assert.False(t, model.ProgrammingLanguage(served).IsValid(),
		"the served value is not a member of the ProgrammingLanguage enum")
	assert.True(t, model.ProgrammingLanguageGo.IsValid())
	assert.NotEqual(t, string(model.ProgrammingLanguageGo), served,
		"the schema member and the served value differ")
	assert.Equal(t, strings.ToLower(string(model.ProgrammingLanguageGo)), strings.ToLower(served),
		"for Go they differ only by case, which is what makes this easy to miss")

	// the languages odiglet can detect that the schema cannot express at all.
	schemaMembers := map[string]bool{}
	for _, member := range model.AllProgrammingLanguage {
		schemaMembers[strings.ToLower(string(member))] = true
	}
	unrepresentable := []string{}
	for _, detected := range []common.ProgrammingLanguage{
		common.JavaProgrammingLanguage, common.GoProgrammingLanguage,
		common.JavascriptProgrammingLanguage, common.PythonProgrammingLanguage,
		common.DotNetProgrammingLanguage, common.NginxProgrammingLanguage,
		common.MySQLProgrammingLanguage, common.PhpProgrammingLanguage,
	} {
		if !schemaMembers[strings.ToLower(string(detected))] {
			unrepresentable = append(unrepresentable, string(detected))
		}
	}
	assert.NotEmpty(t, unrepresentable,
		"at least one detectable language has no ProgrammingLanguage enum member")
}

// TestAgentEnabledReportsThePerContainerDecisionsSortedByName covers the container fan-out
// and the workload-level flag it sits next to.
func TestAgentEnabledReportsThePerContainerDecisionsSortedByName(t *testing.T) {
	id := wrWorkloadID()

	ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrMainIC(func(ic *v1alpha1.InstrumentationConfig) {
			ic.Spec.Containers = []v1alpha1.ContainerAgentConfig{
				{ContainerName: wrSidecarName, AgentEnabled: false, OtelDistroName: ""},
				{ContainerName: wrContainerName, AgentEnabled: true, OtelDistroName: wrReportingDistro},
			}
		}))

	got, err := r.K8sWorkload().AgentEnabled(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.AgentEnabled, "the workload flag comes from the spec, not from the containers")
	require.NotNil(t, got.EnabledStatus)
	require.Len(t, got.Containers, 2)
	assert.Equal(t, []string{wrContainerName, wrSidecarName},
		[]string{got.Containers[0].ContainerName, got.Containers[1].ContainerName})
	assert.True(t, got.Containers[0].AgentEnabled)
	assert.False(t, got.Containers[1].AgentEnabled,
		"each container must report its own decision")

	t.Run("a nil workload id resolves to nothing rather than panicking", func(t *testing.T) {
		got, err := r.K8sWorkload().AgentEnabled(ctx, &model.K8sWorkload{})
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

// TestRolloutReportsTheAgentHashChangeTimeOnlyWhenTheConfigCarriesIt covers the optional
// timestamp and the two nil-guard shapes around it. The timestamp is what the UI uses to
// explain why pods need restarting, so formatting or dropping it changes a user-facing
// instruction.
func TestRolloutReportsTheAgentHashChangeTimeOnlyWhenTheConfigCarriesIt(t *testing.T) {
	id := wrWorkloadID()
	changedAt := metav1.NewTime(time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC))

	t.Run("the config carries a hash change time", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			wrReconciledIC(func(ic *v1alpha1.InstrumentationConfig) {
				ic.Spec.AgentsMetaHashChangedTime = &changedAt
			}))
		got, err := r.K8sWorkload().Rollout(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, wrStr("2026-03-04T05:06:07Z"), got.AgentsMetaHashChangedTime)
		require.NotNil(t, got.RolloutStatus)
		assert.Equal(t, wrStr(string(v1alpha1.WorkloadRolloutReasonRolloutFinished)), got.RolloutStatus.ReasonEnum)
		require.NotNil(t, got.PodsManifestInjectionStatus)
		// the injection status must be derived from the config's own condition, not from
		// the unmarked-workload fallback that a nil config would take.
		assert.Equal(t, wrStr(generatedstatus.PodsManifestInjectionPodsAppliedSuccessfully_Enabled.Title),
			got.PodsManifestInjectionStatus.ReasonEnum)
	})

	t.Run("the config carries no hash change time", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1), wrReconciledIC())
		got, err := r.K8sWorkload().Rollout(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Nil(t, got.AgentsMetaHashChangedTime)
	})

	t.Run("there is no instrumentation config", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1))
		got, err := r.K8sWorkload().Rollout(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got, "rollout is a non-null schema field")
		assert.Nil(t, got.RolloutStatus)
		require.NotNil(t, got.PodsManifestInjectionStatus,
			"the pods injection status is reported even for an unmarked workload")
	})

	t.Run("a nil workload id still yields the unmarked pods injection status", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1), wrReconciledIC())
		got, err := r.K8sWorkload().Rollout(ctx, &model.K8sWorkload{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Nil(t, got.RolloutStatus)
		assert.NotNil(t, got.PodsManifestInjectionStatus)
	})
}

// TestAutoRollbackAndRollbackOccurredReadTheSameStatusField pins the two fields that tell
// the user a workload was automatically rolled back. RollbackOccurred is read by both the
// dedicated resolver and the autoRollback object, so they must never disagree.
func TestAutoRollbackAndRollbackOccurredReadTheSameStatusField(t *testing.T) {
	id := wrWorkloadID()

	for name, rolledBack := range map[string]bool{"a rolled back workload": true, "a healthy workload": false} {
		t.Run(name, func(t *testing.T) {
			ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
				wrDeployment(wrAppNamespace, wrWorkloadName, 1),
				wrReconciledIC(func(ic *v1alpha1.InstrumentationConfig) {
					ic.Status.RollbackOccurred = rolledBack
				}))

			autoRollback, err := r.K8sWorkload().AutoRollback(ctx, &model.K8sWorkload{ID: &id})
			require.NoError(t, err)
			require.NotNil(t, autoRollback)
			require.NotNil(t, autoRollback.AutoRollbackStatus)
			assert.Equal(t, status.RollbackStatus, autoRollback.AutoRollbackStatus.Name)
			assert.Equal(t, rolledBack, autoRollback.RollbackOccurred)

			flag, err := r.K8sWorkload().RollbackOccurred(ctx, &model.K8sWorkload{ID: &id})
			require.NoError(t, err)
			assert.Equal(t, rolledBack, flag)
			assert.Equal(t, autoRollback.RollbackOccurred, flag,
				"the two fields must report the same rollback state")

			if rolledBack {
				assert.Equal(t, wrStr(string(status.AutoRollbackReasonRollbackOccurred)),
					autoRollback.AutoRollbackStatus.ReasonEnum)
			} else {
				assert.NotEqual(t, wrStr(string(status.AutoRollbackReasonRollbackOccurred)),
					autoRollback.AutoRollbackStatus.ReasonEnum)
			}
		})
	}

	t.Run("an unmarked workload has no rollback object and reports false", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1))
		autoRollback, err := r.K8sWorkload().AutoRollback(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.Nil(t, autoRollback)

		flag, err := r.K8sWorkload().RollbackOccurred(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.False(t, flag)
	})

	t.Run("a nil workload id resolves to nothing rather than panicking", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1), wrReconciledIC())
		got, err := r.K8sWorkload().AutoRollback(ctx, &model.K8sWorkload{})
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

// TestContainersMergesTheFourPerContainerSourcesByName covers the merge loop. Each of the
// four spec/status lists names a different container, so a container present in only one of
// them must still appear, and the loop that reads that list cannot be dropped.
func TestContainersMergesTheFourPerContainerSourcesByName(t *testing.T) {
	id := wrWorkloadID()
	ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrMainIC(func(ic *v1alpha1.InstrumentationConfig) {
			ic.Spec.Containers = []v1alpha1.ContainerAgentConfig{
				{ContainerName: "only-in-agent-config", AgentEnabled: true, OtelDistroName: wrReportingDistro},
			}
			ic.Spec.ContainersOverrides = []v1alpha1.ContainerOverride{
				{ContainerName: "only-in-overrides"},
			}
			ic.Status.RuntimeDetailsByContainer = []v1alpha1.RuntimeDetailsByContainer{
				{ContainerName: "only-in-runtime-details", Language: common.GoProgrammingLanguage},
			}
		}))

	got, err := r.K8sWorkload().Containers(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.Len(t, got, 3, "a container named by only one of the sources must still appear")

	byName := map[string]*model.K8sWorkloadContainer{}
	names := []string{}
	for _, c := range got {
		byName[c.ContainerName] = c
		names = append(names, c.ContainerName)
	}
	assert.Equal(t, []string{"only-in-agent-config", "only-in-overrides", "only-in-runtime-details"}, names,
		"containers must be sorted by name")

	require.NotNil(t, byName["only-in-agent-config"].AgentEnabled)
	assert.True(t, byName["only-in-agent-config"].AgentEnabled.AgentEnabled)
	assert.Nil(t, byName["only-in-agent-config"].RuntimeInfo)

	assert.NotNil(t, byName["only-in-overrides"].Overrides)
	assert.Nil(t, byName["only-in-overrides"].AgentEnabled)

	require.NotNil(t, byName["only-in-runtime-details"].RuntimeInfo)
	assert.Equal(t, model.ProgrammingLanguage(common.GoProgrammingLanguage),
		byName["only-in-runtime-details"].RuntimeInfo.Language)
	assert.Nil(t, byName["only-in-runtime-details"].Overrides)

	t.Run("instrumentations are attached per container from its own instances", func(t *testing.T) {
		podName := wrPodName(wrWorkloadName, 0)
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			wrReconciledIC(),
			wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
				agentsMetaHash: wrAgentsMetaHash,
				containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)}}),
			wrInstance(wrInstanceFixture{
				name: "two-libraries", namespace: wrAppNamespace, workloadName: wrWorkloadName,
				podName: podName, containerName: wrContainerName, healthy: wrBool(true),
				components: []v1alpha1.InstrumentationLibraryStatus{
					wrComponent("net/http", wrBool(true)),
					wrComponent("database/sql", wrBool(false)),
					// a non-instrumentation component must be filtered out.
					{Name: "exporter", Type: v1alpha1.InstrumentationLibraryTypeExporter, Healthy: wrBool(true)},
				},
			}))

		got, err := r.K8sWorkload().Containers(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.Len(t, got, 1)
		names := []string{}
		for _, i := range got[0].Instrumentations {
			names = append(names, i.Name)
		}
		assert.Equal(t, []string{"database/sql", "net/http"}, names,
			"only instrumentation libraries are listed, sorted by name")
	})

	t.Run("a nil workload id resolves to nothing rather than panicking", func(t *testing.T) {
		got, err := r.K8sWorkload().Containers(ctx, &model.K8sWorkload{})
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

// TestWorkloadHealthStatusAndNumberOfInstancesComeFromTheWorkloadManifest covers the two
// resolvers that read the cached manifest, including the nil-manifest case a Source CR
// pointing at a deleted workload produces.
func TestWorkloadHealthStatusAndNumberOfInstancesComeFromTheWorkloadManifest(t *testing.T) {
	id := wrWorkloadID()

	t.Run("the deployment exists", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 7), wrReconciledIC())

		health, err := r.K8sWorkload().WorkloadHealthStatus(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, health)
		assert.NotEmpty(t, health.Name)

		instances, err := r.K8sWorkload().NumberOfInstances(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, instances)
		assert.Equal(t, 7, *instances, "the count is the available replicas of the deployment")
	})

	t.Run("the deployment was deleted while the source remains", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrSource(wrAppNamespace, wrWorkloadName, false))

		health, err := r.K8sWorkload().WorkloadHealthStatus(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.Nil(t, health)

		instances, err := r.K8sWorkload().NumberOfInstances(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.Nil(t, instances, "a missing manifest must report no instances rather than zero")
	})

	t.Run("a pre-populated instance count is not recomputed", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 7), wrReconciledIC())
		precomputed := 99
		got, err := r.K8sWorkload().NumberOfInstances(ctx,
			&model.K8sWorkload{ID: &id, NumberOfInstances: &precomputed})
		require.NoError(t, err)
		assert.Same(t, &precomputed, got)
	})
}

// TestProcessesHealthStatusAggregatesTheAgentHealthOfEveryProcess covers the thin resolver
// that forwards to the shared aggregation, including the optional-pod-manifest-injection
// container list it threads through.
func TestProcessesHealthStatusAggregatesTheAgentHealthOfEveryProcess(t *testing.T) {
	id := wrWorkloadID()
	podName := wrPodName(wrWorkloadName, 0)

	t.Run("a healthy agent", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1), wrReconciledIC(),
			wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
				agentsMetaHash: wrAgentsMetaHash,
				containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)}}),
			wrHealthyInstance(podName, wrContainerName, "8080"))

		got, err := r.K8sWorkload().ProcessesHealthStatus(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, model.DesiredStateProgressSuccess, got.Status)
	})

	t.Run("an unhealthy agent", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1), wrReconciledIC(),
			wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
				agentsMetaHash: wrAgentsMetaHash,
				containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)}}),
			wrInstance(wrInstanceFixture{
				name: "unhealthy", namespace: wrAppNamespace, workloadName: wrWorkloadName,
				podName: podName, containerName: wrContainerName, healthy: wrBool(false),
			}))

		got, err := r.K8sWorkload().ProcessesHealthStatus(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.NotEqual(t, model.DesiredStateProgressSuccess, got.Status)
	})
}

// TestTelemetryMetricsReportsNoNumbersUntilTheCollectorHasSeenTheWorkload pins the
// unreported case. A workload the collectors have not reported on must render as unknown
// rather than as zero bytes, which reads as "instrumented but silent".
func TestTelemetryMetricsReportsNoNumbersUntilTheCollectorHasSeenTheWorkload(t *testing.T) {
	ctx, r := wrHarness(t, wrSteadyStateObjects()...)
	id := wrWorkloadID()

	got, err := r.K8sWorkload().TelemetryMetrics(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.Len(t, got, 1, "telemetryMetrics is a non-null list with exactly one element")
	assert.Nil(t, got[0].TotalDataSentBytes)
	assert.Nil(t, got[0].ThroughputBytes)
}

// TestDataStreamNamesMergesTheWorkloadAndNamespaceSources covers the workload-level and
// namespace-level resolvers of the same field name, which read different sources.
func TestDataStreamNamesMergesTheWorkloadAndNamespaceSources(t *testing.T) {
	id := wrWorkloadID()

	t.Run("the workload resolver merges both sources", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			wrSource(wrAppNamespace, wrWorkloadName, false, "from-workload"),
			wrNamespaceSource(wrAppNamespace, false, "from-namespace"))

		got, err := r.K8sWorkload().DataStreamNames(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"from-workload", "from-namespace"}, got)
	})

	// Only the PRIMARY argument of ExtractDataStreamsFromSource can forbid a stream: a
	// "false" label there excludes the stream even when the other source opts in, while
	// a "false" label on the secondary is merely skipped. So the workload source has to be
	// passed first, or a workload loses the ability to opt out of a namespace-wide stream.
	t.Run("a workload can opt out of a stream its namespace opts into", func(t *testing.T) {
		optedOut := wrSource(wrAppNamespace, wrWorkloadName, false, "kept")
		optedOut.Labels[k8sconsts.SourceDataStreamLabelPrefix+"declined"] = "false"

		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			optedOut,
			wrNamespaceSource(wrAppNamespace, false, "declined", "inherited"))

		got, err := r.K8sWorkload().DataStreamNames(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"kept", "inherited"}, got)
		assert.NotContains(t, got, "declined",
			"a stream the workload set to false must not come back via the namespace")
	})

	t.Run("the namespace resolver reads only the namespace source", func(t *testing.T) {
		ctx, r := wrHarnessWithFilter(t, &model.WorkloadFilter{Namespace: wrStr(wrAppNamespace)},
			wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			wrSource(wrAppNamespace, wrWorkloadName, false, "from-workload"),
			wrNamespaceSource(wrAppNamespace, false, "from-namespace"))

		got, err := r.K8sNamespace().DataStreamNames(ctx, &model.K8sNamespace{Name: wrAppNamespace})
		require.NoError(t, err)
		assert.Equal(t, []string{"from-namespace"}, got,
			"a namespace must not inherit its workloads' data streams")
	})

	t.Run("a workload with no sources reports an empty list rather than null", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1))
		got, err := r.K8sWorkload().DataStreamNames(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		assert.NotNil(t, got, "dataStreamNames is a non-null list in the schema")
		assert.Empty(t, got)
	})

	t.Run("a pre-populated list is not recomputed", func(t *testing.T) {
		ctx, r := wrHarness(t, wrNamespaceObject(wrAppNamespace),
			wrDeployment(wrAppNamespace, wrWorkloadName, 1),
			wrSource(wrAppNamespace, wrWorkloadName, false, "from-workload"))
		got, err := r.K8sWorkload().DataStreamNames(ctx,
			&model.K8sWorkload{ID: &id, DataStreamNames: []string{"already-known"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"already-known"}, got)
	})
}

// TestNamespaceMarkedForInstrumentationIsTrueOnlyForAnEnabledNamespaceSource covers the
// three-branch namespace gate. It is what the source-selection screen ticks, so treating a
// disabled namespace source as marked would re-instrument a namespace the user turned off.
func TestNamespaceMarkedForInstrumentationIsTrueOnlyForAnEnabledNamespaceSource(t *testing.T) {
	cases := map[string]struct {
		objects []client.Object
		want    bool
	}{
		"an enabled namespace source":  {objects: []client.Object{wrNamespaceSource(wrAppNamespace, false)}, want: true},
		"a disabled namespace source":  {objects: []client.Object{wrNamespaceSource(wrAppNamespace, true)}, want: false},
		"no namespace source":          {objects: nil, want: false},
		"only a workload source in it": {objects: []client.Object{wrSource(wrAppNamespace, wrWorkloadName, false)}, want: false},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, r := wrHarnessWithFilter(t, &model.WorkloadFilter{Namespace: wrStr(wrAppNamespace)},
				append([]client.Object{
					wrNamespaceObject(wrAppNamespace),
					wrDeployment(wrAppNamespace, wrWorkloadName, 1),
				}, tt.objects...)...)

			got, err := r.K8sNamespace().MarkedForInstrumentation(ctx, &model.K8sNamespace{Name: wrAppNamespace})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestNamespacesOmitsTheNamespacesOdigosIsConfiguredToIgnore covers the Namespaces query
// resolver together with the config-driven filter it applies.
func TestNamespacesOmitsTheNamespacesOdigosIsConfiguredToIgnore(t *testing.T) {
	ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{},
		wrEffectiveConfigMap("configVersion: 1\nignoredNamespaces:\n  - "+wrOtherNamespace+"\n"),
		wrNamespaceObject(wrAppNamespace),
		wrNamespaceObject(wrOtherNamespace),
		wrNamespaceObject(wrOdigosNamespace),
	)

	got, err := r.Query().Namespaces(ctx)
	require.NoError(t, err)
	names := []string{}
	for _, ns := range got {
		names = append(names, ns.Name)
	}
	assert.ElementsMatch(t, []string{wrAppNamespace, wrOdigosNamespace}, names)
	assert.NotContains(t, names, wrOtherNamespace)
}

// TestNamespaceWorkloadsListsOnlyTheWorkloadsOfThatNamespace covers the namespace-scoped
// workload list and the light pre-population it performs. The second namespace is what
// makes a dropped namespace scope observable.
func TestNamespaceWorkloadsListsOnlyTheWorkloadsOfThatNamespace(t *testing.T) {
	ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{},
		wrNamespaceObject(wrAppNamespace),
		wrNamespaceObject(wrOtherNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 4),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrDeployment(wrOtherNamespace, wrOtherName, 1),
		wrSource(wrOtherNamespace, wrOtherName, false),
	)

	got, err := r.K8sNamespace().Workloads(ctx, &model.K8sNamespace{Name: wrAppNamespace})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].ID)
	assert.Equal(t, wrWorkloadName, got[0].ID.Name)
	assert.Equal(t, wrAppNamespace, got[0].ID.Namespace)

	// the three fields populateNamespaceWorkloadLightFields pre-computes so the
	// per-field resolvers short-circuit.
	require.NotNil(t, got[0].NumberOfInstances)
	assert.Equal(t, 4, *got[0].NumberOfInstances)
	assert.Equal(t, []string{wrDataStream}, got[0].DataStreamNames)
	require.NotNil(t, got[0].MarkedForInstrumentation)
	assert.Equal(t, wrBool(true), got[0].MarkedForInstrumentation.MarkedForInstrumentation)

	// and the pod-dependent fields it deliberately does not compute.
	assert.Nil(t, got[0].Conditions, "the namespace list must not load pods or configs")
	assert.Nil(t, got[0].WorkloadOdigosHealthStatus)
}
