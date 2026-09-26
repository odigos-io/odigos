package graph

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Every resolver in workload.resolvers.go that reads the cluster ends in
// `if err != nil { return nil, err }`. Those lines are what keeps a cache read failure
// from rendering as an empty answer: a swallowed pod-list error makes a workload look like
// it has no pods, and a swallowed instance-list error makes a healthy agent look absent.
// Both read as a clean bill of health on a screen whose entire job is to report health.

// wrErrorObjects is the steady-state fixture; every resolver below has real data to return
// when the cluster read succeeds, so a test that passes only because the fixture is empty
// is not possible.
func wrErrorObjects() []client.Object {
	podName := wrPodName(wrWorkloadName, 0)
	return []client.Object{
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrPod(wrPodFixture{
			name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
		}),
		wrHealthyInstance(podName, wrContainerName, "6060"),
	}
}

// wrPodReadingResolvers are the resolvers that load the workload's pods.
func wrPodReadingResolvers(r *Resolver, id *model.K8sWorkloadID) map[string]func(context.Context) (interface{}, error) {
	workload := func() *model.K8sWorkload { return &model.K8sWorkload{ID: id} }
	return map[string]func(context.Context) (interface{}, error){
		"pods": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().Pods(ctx, workload())
		},
		"podsHealthStatus": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().PodsHealthStatus(ctx, workload())
		},
		"podsOdigosHealthStatus": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().PodsOdigosHealthStatus(ctx, workload())
		},
		"podsAgentInjectionStatus": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().PodsAgentInjectionStatus(ctx, workload())
		},
		"conditions": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().Conditions(ctx, workload())
		},
		"workloadOdigosHealthStatus": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().WorkloadOdigosHealthStatus(ctx, workload())
		},
		"rollout": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().Rollout(ctx, workload())
		},
		"rollout.podsManifestInjectionOverview": func(ctx context.Context) (interface{}, error) {
			chained := wrFieldChain(ctx, wrWorkloadResult(), nil, nil)
			return r.K8sWorkloadRollout().PodsManifestInjectionOverview(chained, &model.K8sWorkloadRollout{})
		},
		"telemetryMetrics.expectingTelemetry": func(ctx context.Context) (interface{}, error) {
			chained := wrFieldChain(ctx, wrWorkloadResult(), nil, nil, nil)
			return r.K8sWorkloadTelemetryMetrics().ExpectingTelemetry(chained, &model.K8sWorkloadTelemetryMetrics{})
		},
	}
}

// wrInstanceReadingResolvers are the resolvers that load InstrumentationInstances.
func wrInstanceReadingResolvers(r *Resolver, id *model.K8sWorkloadID) map[string]func(context.Context) (interface{}, error) {
	workload := func() *model.K8sWorkload { return &model.K8sWorkload{ID: id} }
	return map[string]func(context.Context) (interface{}, error){
		"pods": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().Pods(ctx, workload())
		},
		"podsOdigosHealthStatus": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().PodsOdigosHealthStatus(ctx, workload())
		},
		"processesHealthStatus": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().ProcessesHealthStatus(ctx, workload())
		},
		"containers": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().Containers(ctx, workload())
		},
		"conditions": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().Conditions(ctx, workload())
		},
		"workloadOdigosHealthStatus": func(ctx context.Context) (interface{}, error) {
			return r.K8sWorkload().WorkloadOdigosHealthStatus(ctx, workload())
		},
		"podContainer.processes": func(ctx context.Context) (interface{}, error) {
			chained := wrFieldChain(ctx, wrWorkloadResult(), nil,
				wrPodResult(wrPodName(wrWorkloadName, 0)), nil, nil, nil)
			return r.K8sWorkloadPodContainer().Processes(chained,
				&model.K8sWorkloadPodContainer{ContainerName: wrContainerName})
		},
	}
}

func TestEveryPodReadingResolverPropagatesAClusterReadFailure(t *testing.T) {
	id := wrWorkloadID()

	// first prove each resolver returns real data when the cluster read succeeds, so the
	// error assertions below cannot pass on an empty fixture.
	healthyCtx, healthyResolver := wrHarness(t, wrErrorObjects()...)
	for name, resolve := range wrPodReadingResolvers(healthyResolver, &id) {
		got, err := resolve(healthyCtx)
		require.NoError(t, err, "%s failed against a healthy cluster", name)
		assert.NotNil(t, got, "%s returned nothing against a healthy cluster", name)
	}

	brokenCtx, brokenResolver := wrFailingHarness(t, wrInterceptors{failPodList: true}, wrErrorObjects()...)
	resolvers := wrPodReadingResolvers(brokenResolver, &id)
	for name, resolve := range resolvers {
		t.Run(name, func(t *testing.T) {
			got, err := resolve(brokenCtx)
			require.Error(t, err, "a failed pod list must not be reported as an empty pod list")
			assert.True(t, apierrors.IsServiceUnavailable(err),
				"the kubernetes error must reach the caller unwrapped, got %v", err)
			assert.Nil(t, got)
		})
	}
	require.Len(t, resolvers, 9)
}

func TestEveryInstanceReadingResolverPropagatesAClusterReadFailure(t *testing.T) {
	id := wrWorkloadID()

	healthyCtx, healthyResolver := wrHarness(t, wrErrorObjects()...)
	for name, resolve := range wrInstanceReadingResolvers(healthyResolver, &id) {
		got, err := resolve(healthyCtx)
		require.NoError(t, err, "%s failed against a healthy cluster", name)
		assert.NotNil(t, got, "%s returned nothing against a healthy cluster", name)
	}

	brokenCtx, brokenResolver := wrFailingHarness(t, wrInterceptors{failInstanceList: true}, wrErrorObjects()...)
	resolvers := wrInstanceReadingResolvers(brokenResolver, &id)
	for name, resolve := range resolvers {
		t.Run(name, func(t *testing.T) {
			got, err := resolve(brokenCtx)
			require.Error(t, err, "a failed instance list must not be reported as a healthy agent")
			assert.True(t, apierrors.IsServiceUnavailable(err),
				"the kubernetes error must reach the caller unwrapped, got %v", err)
			assert.Nil(t, got)
		})
	}
	require.Len(t, resolvers, 7)
}

// TestAFailedPodReadFailsTheWholeGraphqlFieldRatherThanReturningPartialData checks the
// consequence through the real executor: the field is nulled and an error is reported,
// rather than the operation succeeding with a plausible-looking empty answer.
func TestAFailedPodReadFailsTheWholeGraphqlFieldRatherThanReturningPartialData(t *testing.T) {
	ctx, r := wrFailingHarness(t, wrInterceptors{failPodList: true}, wrErrorObjects()...)

	data, errs := wrExecuteRaw(t, ctx, r, `
		query($filter: WorkloadFilter) {
			workloads(filter: $filter) { id { name } pods { podName } }
		}`, wrSingleWorkloadVars())

	require.NotEmpty(t, errs, "the operation must report the cluster failure")
	assert.Contains(t, errs[0], "informer cache is not synced")
	// pods is a nullable list, so gqlgen nulls the field rather than the whole workload.
	if workloads, ok := data["workloads"].([]interface{}); ok && len(workloads) == 1 {
		assert.Nil(t, wrObject(t, workloads[0])["pods"],
			"a failed read must not surface as an empty pod list")
	}
}

// TestNamespaceResolversPropagateASourceReadFailure covers the two k8sNamespace resolvers'
// error paths. Reporting a namespace as not marked for instrumentation because the Source
// read failed would invite the operator to re-enable something already enabled.
func TestNamespaceResolversPropagateASourceReadFailure(t *testing.T) {
	objects := []client.Object{
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrNamespaceSource(wrAppNamespace, false, wrDataStream),
	}

	ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{failSourceList: true}, objects...)

	// Sources are loaded lazily by these resolvers, so the Namespaces query runs first —
	// exactly as gqlgen resolves it, parent before children — and only then does the
	// broken Source list reach each field resolver's own error branch.
	namespaces, err := r.Query().Namespaces(ctx)
	require.NoError(t, err, "the namespace list itself does not read Sources")
	require.Len(t, namespaces, 1)

	marked, err := r.K8sNamespace().MarkedForInstrumentation(ctx, &model.K8sNamespace{Name: wrAppNamespace})
	require.Error(t, err)
	assert.False(t, marked, "a failed read must not be reported as marked for instrumentation")

	names, err := r.K8sNamespace().DataStreamNames(ctx, &model.K8sNamespace{Name: wrAppNamespace})
	require.Error(t, err)
	assert.Empty(t, names)

	workloads, err := r.K8sNamespace().Workloads(ctx, &model.K8sNamespace{Name: wrAppNamespace})
	require.Error(t, err)
	assert.Nil(t, workloads)
}

// TestContainersMergesTheWorkloadCollectorConfigList covers the fourth of the five
// per-container sources the Containers resolver merges, which no other test names.
func TestContainersMergesTheWorkloadCollectorConfigList(t *testing.T) {
	id := wrWorkloadID()
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrMainIC(func(ic *v1alpha1.InstrumentationConfig) {
			ic.Spec.WorkloadCollectorConfig = []commonapi.ContainerCollectorConfig{
				{ContainerName: "only-in-collector-config"},
			}
		}))

	got, err := r.K8sWorkload().Containers(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)

	byName := map[string]*model.K8sWorkloadContainer{}
	for _, c := range got {
		byName[c.ContainerName] = c
	}
	require.Contains(t, byName, "only-in-collector-config",
		"a container named only by the collector config must still appear")
	assert.NotNil(t, byName["only-in-collector-config"].CollectorConfig)
	assert.Nil(t, byName["only-in-collector-config"].AgentEnabled)

	// and the container named by the agent config keeps its own collector config absent.
	require.Contains(t, byName, wrContainerName)
	assert.Nil(t, byName[wrContainerName].CollectorConfig)
	assert.NotNil(t, byName[wrContainerName].AgentEnabled)
}

// TestProcessesListsOnlyInstrumentationLibrariesAndSortsProcessesByPid covers the
// component-type filter and the pid ordering. The pid sort is what keeps the process list
// from reshuffling between refreshes, and it reads an agent-emitted attribute name that
// nothing links to the agent at compile time.
func TestProcessesListsOnlyInstrumentationLibrariesAndSortsProcessesByPid(t *testing.T) {
	podName := wrPodName(wrWorkloadName, 0)
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrPod(wrPodFixture{name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)}}),
		// two processes in the container, seeded with the higher pid first.
		wrInstance(wrInstanceFixture{
			name: "high-pid", namespace: wrAppNamespace, workloadName: wrWorkloadName,
			podName: podName, containerName: wrContainerName, pid: "900", healthy: wrBool(true),
			components: []v1alpha1.InstrumentationLibraryStatus{
				wrComponent("net/http", wrBool(true)),
				{Name: "otlp-exporter", Type: v1alpha1.InstrumentationLibraryTypeExporter, Healthy: wrBool(true)},
			},
		}),
		wrInstance(wrInstanceFixture{
			name: "low-pid", namespace: wrAppNamespace, workloadName: wrWorkloadName,
			podName: podName, containerName: wrContainerName, pid: "100", healthy: wrBool(true),
			// declared out of alphabetical order to exercise the library sort.
			components: []v1alpha1.InstrumentationLibraryStatus{
				wrComponent("net/http", wrBool(true)),
				wrComponent("database/sql", wrBool(true)),
			},
		}),
	)

	chained := wrFieldChain(ctx, wrWorkloadResult(), nil, wrPodResult(podName), nil, nil, nil)
	got, err := r.K8sWorkloadPodContainer().Processes(chained,
		&model.K8sWorkloadPodContainer{ContainerName: wrContainerName})
	require.NoError(t, err)
	require.Len(t, got, 2)

	pidOf := func(p *model.K8sWorkloadPodContainerProcess) string {
		for _, attr := range p.IdentifyingAttributes {
			if attr.Name == processAttributeNamePid {
				return attr.Value
			}
		}
		t.Fatalf("no %s attribute on the process", processAttributeNamePid)
		return ""
	}
	assert.Equal(t, []string{"100", "900"}, []string{pidOf(got[0]), pidOf(got[1])},
		"processes must be ordered by pid so the list does not reshuffle between refreshes")

	// the identifying attributes are sorted by name, so the executable name precedes the
	// pid. The UI renders them in the order it receives them.
	for _, p := range got {
		names := []string{}
		for _, attr := range p.IdentifyingAttributes {
			names = append(names, attr.Name)
		}
		assert.Equal(t, []string{"process.executable.name", processAttributeNamePid}, names,
			"identifying attributes must be sorted by name")
	}

	assert.Equal(t, []string{"database/sql", "net/http"}, wrInstrumentationNames(got[0]),
		"instrumentation libraries must be sorted by name")
	assert.Equal(t, []string{"net/http"}, wrInstrumentationNames(got[1]),
		"a non-instrumentation component must not be listed as an instrumentation")
}

func wrInstrumentationNames(p *model.K8sWorkloadPodContainerProcess) []string {
	names := []string{}
	for _, i := range p.Instrumentations {
		names = append(names, i.Name)
	}
	return names
}

// TestTheWorkloadQueriesRefuseAnIgnoredNamespaceInsteadOfReturningNothing covers the guard
// in LoadWorkloadsWithFilter as it surfaces through the query resolver. Silently returning
// an empty list would read as "this namespace has no workloads" rather than "odigos is
// configured to ignore it".
func TestTheWorkloadQueriesRefuseAnIgnoredNamespaceInsteadOfReturningNothing(t *testing.T) {
	ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{},
		wrEffectiveConfigMap("configVersion: 1\nignoredNamespaces:\n  - "+wrOtherNamespace+"\n"),
		wrNamespaceObject(wrOtherNamespace),
		wrDeployment(wrOtherNamespace, wrOtherName, 1),
	)

	got, err := r.Query().Workloads(ctx, &model.WorkloadFilter{Namespace: wrStr(wrOtherNamespace)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), wrOtherNamespace,
		"the error must name the namespace it refused")
	assert.Nil(t, got)
}

// TestTheNamespacesQueryFailsWhenTheOdigosConfigIsUnreadable covers the LoadConfig error
// path. An empty namespace list from a misconfigured installation would look like an empty
// cluster and invite the operator to go looking in the wrong place.
func TestTheNamespacesQueryFailsWhenTheOdigosConfigIsUnreadable(t *testing.T) {
	// the effective-config ConfigMap lives in a different namespace than CURRENT_NS.
	ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: consts.OdigosEffectiveConfigName, Namespace: wrOtherNamespace,
		}},
		wrNamespaceObject(wrAppNamespace),
	)

	got, err := r.Query().Namespaces(ctx)
	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err), "the kubernetes error must reach the caller, got %v", err)
	assert.Nil(t, got)
}
