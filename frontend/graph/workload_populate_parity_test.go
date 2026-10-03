package graph

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The workloads list resolver pre-computes fields eagerly (populateWorkloadFields) while
// the single-workload page resolves the same fields lazily. The two are separate
// implementations of the same answer, so they can drift: a fix applied to one leaves the
// list view and the detail view disagreeing about the same workload. The tests in this
// file drive both paths against one fake cluster and pin where they agree — and, just as
// importantly, where they already do not.

// wpParity runs populateWorkloadFields on one workload and hands back a second, empty
// workload plus the resolver, so each lazy resolver can be invoked without its
// short-circuit guard firing.
func wpParity(t *testing.T, f wpFixture) (eager *model.K8sWorkload, lazy *model.K8sWorkload, r *k8sWorkloadResolver, ctx context.Context) {
	t.Helper()
	resolver, ctx := wpHarness(t, f)

	eagerID := wpWorkloadID()
	eager = &model.K8sWorkload{ID: &eagerID}
	(&queryResolver{resolver}).populateWorkloadFields(ctx, loaders.For(ctx), eager, f.tier)

	lazyID := wpWorkloadID()
	return eager, &model.K8sWorkload{ID: &lazyID}, &k8sWorkloadResolver{resolver}, ctx
}

// Every field populateWorkloadFields assigns exists so the matching resolver can return
// it without touching the cluster again. Returning the identical pointer is the proof
// that the short-circuit fired: a recomputed value would be a different allocation.
func TestTheEagerlyPopulatedFieldsShortCircuitTheirResolvers(t *testing.T) {
	eager, _, r, ctx := wpParity(t, wpSteadyStateFixture())

	serviceName, err := r.ServiceName(ctx, eager)
	require.NoError(t, err)
	assert.Same(t, eager.ServiceName, serviceName)

	health, err := r.WorkloadOdigosHealthStatus(ctx, eager)
	require.NoError(t, err)
	assert.Same(t, eager.WorkloadOdigosHealthStatus, health)

	conditions, err := r.Conditions(ctx, eager)
	require.NoError(t, err)
	assert.Same(t, eager.Conditions, conditions)

	marked, err := r.MarkedForInstrumentation(ctx, eager)
	require.NoError(t, err)
	assert.Same(t, eager.MarkedForInstrumentation, marked)

	runtimeInfo, err := r.RuntimeInfo(ctx, eager)
	require.NoError(t, err)
	assert.Same(t, eager.RuntimeInfo, runtimeInfo)

	injection, err := r.PodsAgentInjectionStatus(ctx, eager)
	require.NoError(t, err)
	assert.Same(t, eager.PodsAgentInjectionStatus, injection)

	numberOfInstances, err := r.NumberOfInstances(ctx, eager)
	require.NoError(t, err)
	assert.Same(t, eager.NumberOfInstances, numberOfInstances)

	dataStreamNames, err := r.DataStreamNames(ctx, eager)
	require.NoError(t, err)
	assert.Equal(t, eager.DataStreamNames, dataStreamNames)
}

// containers and rollbackOccurred are pre-computed by populateWorkloadFields but their
// resolvers have no short-circuit guard, so gqlgen recomputes both for every workload in
// the list — exactly the per-workload cluster work the eager pass exists to avoid, and
// for containers it is a second round of instrumentation-instance lookups per container.
// Pinned as characterisation: if a guard is added later this test fails and should be
// replaced by a short-circuit assertion above.
func TestContainersAndRollbackOccurredArePrecomputedButNeverShortCircuited(t *testing.T) {
	eager, _, r, ctx := wpParity(t, wpSteadyStateFixture())
	require.NotEmpty(t, eager.Containers, "precondition: the eager pass computed the containers")

	containers, err := r.Containers(ctx, eager)
	require.NoError(t, err)
	require.Len(t, containers, len(eager.Containers))
	assert.NotSame(t, eager.Containers[0], containers[0],
		"the resolver rebuilt the container list instead of reusing the pre-computed one")
}

// The same workload must get the same service name whether it is rendered in the list or
// on its own page.
func TestServiceNameAgreesBetweenTheEagerAndLazyPaths(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mutate  func(*wpFixture)
		wantNil bool
	}{
		{"from the instrumentation config", func(*wpFixture) {}, false},
		{
			name: "from the source when the instrumentation config is gone",
			mutate: func(f *wpFixture) {
				f.ic = nil
				f.source = wpWorkloadSource("service-from-source", true)
			},
		},
		{
			name:    "absent when neither carries a name",
			mutate:  func(f *wpFixture) { f.ic = nil; f.source = nil },
			wantNil: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := wpSteadyStateFixture()
			tt.mutate(&f)

			eager, lazy, r, ctx := wpParity(t, f)

			got, err := r.ServiceName(ctx, lazy)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, eager.ServiceName)
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, eager.ServiceName)
			require.NotNil(t, got)
			assert.Equal(t, *eager.ServiceName, *got)
		})
	}
}

func TestMarkedForInstrumentationAgreesBetweenTheEagerAndLazyPaths(t *testing.T) {
	for _, tt := range []struct {
		name      string
		hasSource bool
		disabled  bool
	}{
		{name: "an enabled source", hasSource: true},
		{name: "a disabled source", hasSource: true, disabled: true},
		{name: "no source"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := wpSteadyStateFixture()
			f.source = nil
			if tt.hasSource {
				f.source = wpWorkloadSource("", tt.disabled)
			}

			eager, lazy, r, ctx := wpParity(t, f)

			got, err := r.MarkedForInstrumentation(ctx, lazy)
			require.NoError(t, err)
			assert.Equal(t, eager.MarkedForInstrumentation, got)
		})
	}
}

func TestRuntimeInfoAgreesBetweenTheEagerAndLazyPaths(t *testing.T) {
	eager, lazy, r, ctx := wpParity(t, wpSteadyStateFixture())

	got, err := r.RuntimeInfo(ctx, lazy)
	require.NoError(t, err)
	assert.Equal(t, eager.RuntimeInfo, got)
}

func TestNumberOfInstancesAndDataStreamNamesAgreeBetweenTheEagerAndLazyPaths(t *testing.T) {
	eager, lazy, r, ctx := wpParity(t, wpSteadyStateFixture())

	instances, err := r.NumberOfInstances(ctx, lazy)
	require.NoError(t, err)
	require.NotNil(t, eager.NumberOfInstances)
	require.NotNil(t, instances)
	assert.Equal(t, *eager.NumberOfInstances, *instances)

	names, err := r.DataStreamNames(ctx, lazy)
	require.NoError(t, err)
	assert.Equal(t, eager.DataStreamNames, names)
}

func TestPodsAgentInjectionStatusAgreesBetweenTheEagerAndLazyPaths(t *testing.T) {
	eager, lazy, r, ctx := wpParity(t, wpSteadyStateFixture())

	got, err := r.PodsAgentInjectionStatus(ctx, lazy)
	require.NoError(t, err)
	assert.Equal(t, eager.PodsAgentInjectionStatus, got)
}

// Six of the seven conditions agree. autoRollback does not: the lazy resolver computes
// it, the eager pass never assigns it, so the workloads list reports no auto-rollback
// condition for any workload while the detail page does. Characterised rather than
// fixed — the assertion below fails the moment either side changes.
func TestConditionsAgreeBetweenTheEagerAndLazyPathsExceptForAutoRollback(t *testing.T) {
	eager, lazy, r, ctx := wpParity(t, wpSteadyStateFixture())

	got, err := r.Conditions(ctx, lazy)
	require.NoError(t, err)
	require.NotNil(t, eager.Conditions)
	require.NotNil(t, got)

	assert.Equal(t, eager.Conditions.RuntimeDetection, got.RuntimeDetection)
	assert.Equal(t, eager.Conditions.AgentInjectionEnabled, got.AgentInjectionEnabled)
	assert.Equal(t, eager.Conditions.Rollout, got.Rollout)
	assert.Equal(t, eager.Conditions.PodsManifestInjection, got.PodsManifestInjection)
	assert.Equal(t, eager.Conditions.AgentInjected, got.AgentInjected)
	assert.Equal(t, eager.Conditions.ProcessesAgentHealth, got.ProcessesAgentHealth)
	assert.Equal(t, eager.Conditions.ExpectingTelemetry, got.ExpectingTelemetry)

	assert.Nil(t, eager.Conditions.AutoRollback, "the eager pass does not compute autoRollback")
	assert.NotNil(t, got.AutoRollback, "the lazy resolver does compute autoRollback")
}

// The eager pass omits the conditions entirely for an uninstrumented workload; the lazy
// resolver returns them. The list view therefore renders no condition rows for a
// disabled workload while its detail page renders a full set.
func TestAnUninstrumentedWorkloadGetsConditionsFromTheLazyPathOnly(t *testing.T) {
	f := wpSteadyStateFixture()
	f.ic = nil
	f.source = wpWorkloadSource("", true)

	eager, lazy, r, ctx := wpParity(t, f)

	assert.Nil(t, eager.Conditions)

	got, err := r.Conditions(ctx, lazy)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.NotNil(t, got.PodsManifestInjection)
	assert.NotNil(t, got.AgentInjected)
}

// The aggregated health badge is the single most visible thing on the workloads list,
// and the two paths aggregate different inputs: the eager pass feeds the rollout and
// agent-injected conditions into the severity aggregation, the lazy resolver does not.
// For a workload in steady state both land on the same green result, which is what this
// pins; the divergence below is what makes that agreement fragile.
func TestWorkloadOdigosHealthStatusAgreesInSteadyState(t *testing.T) {
	eager, lazy, r, ctx := wpParity(t, wpSteadyStateFixture())

	got, err := r.WorkloadOdigosHealthStatus(ctx, lazy)
	require.NoError(t, err)
	assert.Equal(t, eager.WorkloadOdigosHealthStatus, got)
	require.NotNil(t, got)
	assert.Equal(t, model.DesiredStateProgressSuccess, got.Status, "precondition: the fixture is fully healthy")
}

// When a rollout has not landed the two paths disagree: the eager pass aggregates the
// agent-injected condition and reports the rollout problem, while the lazy resolver
// leaves that condition out of the aggregation entirely and reports something milder.
// Same workload, two different badges depending on which screen you are looking at.
func TestWorkloadOdigosHealthStatusDivergesWhenTheAgentIsNotInjected(t *testing.T) {
	f := wpSteadyStateFixture()
	f.agentsMetaHash = "hash-v0"

	eager, lazy, r, ctx := wpParity(t, f)

	got, err := r.WorkloadOdigosHealthStatus(ctx, lazy)
	require.NoError(t, err)
	require.NotNil(t, eager.WorkloadOdigosHealthStatus)
	require.NotNil(t, got)

	require.NotNil(t, eager.Conditions)
	require.NotNil(t, eager.Conditions.AgentInjected)
	assert.Equal(t, eager.Conditions.AgentInjected.Status, eager.WorkloadOdigosHealthStatus.Status,
		"the eager pass surfaces the agent-injection problem as the workload health")
	assert.NotEqual(t, eager.WorkloadOdigosHealthStatus.Status, got.Status,
		"the lazy resolver never sees the agent-injected condition, so it reports a different severity")
}

// For an uninstrumented workload the eager pass explains *why* the source is not
// instrumented, while the lazy resolver hardcodes a generic sentence. The specific
// message is the one the user needs, and which one they get depends on the screen.
func TestTheDisabledHealthMessageDivergesBetweenTheEagerAndLazyPaths(t *testing.T) {
	f := wpSteadyStateFixture()
	f.ic = nil
	f.source = wpWorkloadSource("", true)
	// A user who disabled a source and let odigos roll the pods: no agent left on the
	// pods, so nothing outranks the "disabled" condition in the severity aggregation.
	f.agentsMetaHash = ""

	eager, lazy, r, ctx := wpParity(t, f)

	got, err := r.WorkloadOdigosHealthStatus(ctx, lazy)
	require.NoError(t, err)
	require.NotNil(t, eager.WorkloadOdigosHealthStatus)
	require.NotNil(t, got)

	assert.Equal(t, model.DesiredStateProgressDisabled, eager.WorkloadOdigosHealthStatus.Status)
	require.NotNil(t, eager.MarkedForInstrumentation)
	assert.Equal(t, eager.MarkedForInstrumentation.Message, eager.WorkloadOdigosHealthStatus.Message)
	assert.Equal(t, "workload is not marked for instrumentation", got.Message,
		"the lazy resolver reports the generic message instead of the source decision")
	assert.NotEqual(t, eager.WorkloadOdigosHealthStatus.Message, got.Message)
}
