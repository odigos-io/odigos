package graph

import (
	"context"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Three resolvers in workload.resolvers.go cannot be given their parent object by gqlgen,
// so they climb graphql.GetFieldContext(ctx) a hardcoded number of levels and type-assert
// the ancestor's Result:
//
//	K8sWorkloadPodContainer.processes                 -> 3 levels to the pod, then 2 more to the workload
//	K8sWorkloadRollout.podsManifestInjectionOverview  -> 2 levels to the workload
//	K8sWorkloadTelemetryMetrics.expectingTelemetry    -> 3 levels to the workload
//
// Nothing links those depths to the schema at compile time. The two that differ do so only
// because `rollout` is a single object while `telemetryMetrics` is a list, which adds one
// element-level context. Changing a field between object and list, wrapping one in a new
// type, or renaming a field so gqlgen re-nests it silently turns every one of these into
// the "parent is not a workload" error branch: the pod/process view, the rollout overview
// and the telemetry badge all go blank with no compiler or schema error.
//
// Every test here therefore drives the real generated executor, so the parent chain under
// assertion is the one gqlgen actually builds.

const wrParentContextQuery = `
	query($filter: WorkloadFilter) {
		workloads(filter: $filter) {
			id { namespace kind name }
			pods {
				podName
				containers {
					containerName
					processes { healthy identifyingAttributes { name value } }
				}
			}
			rollout {
				podsManifestInjectionOverview { totalPods totalAgentAppliedPods totalAgentNotAppliedPods agentAppliedOk agentNotAppliedOk }
			}
			telemetryMetrics {
				expectingTelemetry { isExpectingTelemetry telemetryObservedStatus { reasonEnum } }
			}
		}
	}`

// wrProcessPids extracts the process.pid identifying attribute of every process the
// container resolved. The pid is what identifies which InstrumentationInstance the
// resolver actually looked up.
func wrProcessPids(t *testing.T, container map[string]interface{}) []string {
	t.Helper()
	pids := []string{}
	for _, rawProcess := range wrList(t, container, "processes") {
		process := wrObject(t, rawProcess)
		for _, rawAttr := range wrList(t, process, "identifyingAttributes") {
			attr := wrObject(t, rawAttr)
			if wrString(t, attr, "name") == processAttributeNamePid {
				pids = append(pids, wrString(t, attr, "value"))
			}
		}
	}
	return pids
}

// TestProcessesAreLookedUpByTheAncestorPodAndTheContainerBeingResolved is the core
// identity assertion. Processes builds a loaders.PodContainerId out of three strings taken
// from three different places: the namespace from the workload five levels up, the pod name
// from the pod three levels up, and the container name from the object being resolved.
// A fixture with one pod and one container cannot tell any of those apart.
func TestProcessesAreLookedUpByTheAncestorPodAndTheContainerBeingResolved(t *testing.T) {
	podA := wrPodName(wrWorkloadName, 0)
	podB := wrPodName(wrWorkloadName, 1)

	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 2),
		wrReconciledIC(),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrPod(wrPodFixture{
			name: podA, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers: []wrContainerFixture{
				wrHealthyContainer(wrContainerName, wrReportingDistro),
				wrHealthyContainer(wrSidecarName, wrReportingDistro),
			},
		}),
		wrPod(wrPodFixture{
			name: podB, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers: []wrContainerFixture{
				wrHealthyContainer(wrContainerName, wrReportingDistro),
				wrHealthyContainer(wrSidecarName, wrReportingDistro),
			},
		}),
		// four instances, one per (pod, container), each with a unique pid.
		wrHealthyInstance(podA, wrContainerName, "1001"),
		wrHealthyInstance(podA, wrSidecarName, "1002"),
		wrHealthyInstance(podB, wrContainerName, "2001"),
		wrHealthyInstance(podB, wrSidecarName, "2002"),
	)

	data := wrExecute(t, ctx, r, wrParentContextQuery, wrSingleWorkloadVars())
	pods := wrList(t, wrOnlyWorkload(t, data), "pods")
	require.Len(t, pods, 2, "both pods must be resolved")

	// pods and containers are both sorted by name, so the expected pid is positional.
	wantPids := map[string]map[string]string{
		podA: {wrContainerName: "1001", wrSidecarName: "1002"},
		podB: {wrContainerName: "2001", wrSidecarName: "2002"},
	}
	seen := 0
	for _, rawPod := range pods {
		pod := wrObject(t, rawPod)
		podName := wrString(t, pod, "podName")
		containers := wrList(t, pod, "containers")
		require.Len(t, containers, 2)
		for _, rawContainer := range containers {
			container := wrObject(t, rawContainer)
			containerName := wrString(t, container, "containerName")
			assert.Equal(t, []string{wantPids[podName][containerName]}, wrProcessPids(t, container),
				"pod %q container %q resolved the wrong process", podName, containerName)
			seen++
		}
	}
	require.Equal(t, 4, seen, "every (pod, container) pair must have been asserted")
}

// TestTheNestedResolversWorkFromBothRootPathsIntoAWorkload pins that the climb is relative
// to the object being resolved rather than to the root of the operation. K8sNamespace.workloads
// nests a workload two levels deeper than Query.workloads, so a resolver that counted from
// the root would work on exactly one of these two paths.
func TestTheNestedResolversWorkFromBothRootPathsIntoAWorkload(t *testing.T) {
	podName := wrPodName(wrWorkloadName, 0)
	objects := []client.Object{
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrSimplePod(0, wrHealthyContainer(wrContainerName, wrReportingDistro)),
		wrHealthyInstance(podName, wrContainerName, "7777"),
	}

	viaWorkloadsQuery := func(t *testing.T) map[string]interface{} {
		ctx, r := wrHarness(t, objects...)
		return wrOnlyWorkload(t, wrExecute(t, ctx, r, wrParentContextQuery, wrSingleWorkloadVars()))
	}

	viaNamespacesQuery := func(t *testing.T) map[string]interface{} {
		ctx, r, _ := wrUnloadedHarness(t, wrInterceptors{}, objects...)
		data := wrExecute(t, ctx, r, `
			query {
				namespaces {
					name
					workloads {
						id { namespace kind name }
						pods {
							podName
							containers { containerName processes { healthy identifyingAttributes { name value } } }
						}
						rollout { podsManifestInjectionOverview { totalPods totalAgentAppliedPods totalAgentNotAppliedPods agentAppliedOk agentNotAppliedOk } }
						telemetryMetrics { expectingTelemetry { isExpectingTelemetry telemetryObservedStatus { reasonEnum } } }
					}
				}
			}`, nil)

		namespaces := wrList(t, data, "namespaces")
		require.Len(t, namespaces, 1, "only the app namespace exists in this fixture")
		workloads := wrList(t, wrObject(t, namespaces[0]), "workloads")
		require.Len(t, workloads, 1)
		return wrObject(t, workloads[0])
	}

	for name, resolve := range map[string]func(*testing.T) map[string]interface{}{
		"Query.workloads":        viaWorkloadsQuery,
		"K8sNamespace.workloads": viaNamespacesQuery,
	} {
		t.Run(name, func(t *testing.T) {
			w := resolve(t)

			pods := wrList(t, w, "pods")
			require.Len(t, pods, 1)
			containers := wrList(t, wrObject(t, pods[0]), "containers")
			require.Len(t, containers, 1)
			assert.Equal(t, []string{"7777"}, wrProcessPids(t, wrObject(t, containers[0])),
				"processes must resolve through this root path too")

			overview := wrChild(t, wrChild(t, w, "rollout"), "podsManifestInjectionOverview")
			assert.EqualValues(t, 1, overview["totalPods"])
			assert.EqualValues(t, 1, overview["totalAgentAppliedPods"])
			assert.EqualValues(t, 0, overview["totalAgentNotAppliedPods"])

			telemetry := wrList(t, w, "telemetryMetrics")
			require.Len(t, telemetry, 1)
			expecting := wrChild(t, wrObject(t, telemetry[0]), "expectingTelemetry")
			assert.Equal(t, true, expecting["isExpectingTelemetry"])
		})
	}
}

// TestTheNestedResolversReadTheirOwnWorkloadWhenSeveralAreResolvedTogether is the
// anti-crosstalk assertion: with two workloads in one response, a resolver that grabbed
// the wrong ancestor, or cached one workload id across the batch, would report one
// workload's pod counts for the other.
func TestTheNestedResolversReadTheirOwnWorkloadWhenSeveralAreResolvedTogether(t *testing.T) {
	ctx, r := wrHarnessWithFilter(t,
		&model.WorkloadFilter{Namespace: wrStr(wrAppNamespace)},
		wrNamespaceObject(wrAppNamespace),
		// the main workload has one injected pod.
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrSimplePod(0, wrHealthyContainer(wrContainerName, wrReportingDistro)),
		// the other workload has two pods, neither carrying the agent hash label.
		wrDeployment(wrAppNamespace, wrOtherName, 2),
		wrIC(wrAppNamespace, wrOtherName),
		wrSource(wrAppNamespace, wrOtherName, false),
		wrPod(wrPodFixture{
			name: wrPodName(wrOtherName, 0), workloadName: wrOtherName, namespace: wrAppNamespace,
			containers: []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
		}),
		wrPod(wrPodFixture{
			name: wrPodName(wrOtherName, 1), workloadName: wrOtherName, namespace: wrAppNamespace,
			containers: []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
		}),
	)

	data := wrExecute(t, ctx, r, wrParentContextQuery,
		map[string]interface{}{"filter": map[string]interface{}{"namespace": wrAppNamespace}})

	overviewByWorkload := map[string]map[string]interface{}{}
	for _, raw := range wrList(t, data, "workloads") {
		w := wrObject(t, raw)
		name := wrString(t, wrChild(t, w, "id"), "name")
		overviewByWorkload[name] = wrChild(t, wrChild(t, w, "rollout"), "podsManifestInjectionOverview")
	}
	require.Len(t, overviewByWorkload, 2)

	assert.EqualValues(t, 1, overviewByWorkload[wrWorkloadName]["totalPods"])
	assert.EqualValues(t, 1, overviewByWorkload[wrWorkloadName]["totalAgentAppliedPods"])
	assert.EqualValues(t, 0, overviewByWorkload[wrWorkloadName]["totalAgentNotAppliedPods"])
	assert.Equal(t, true, overviewByWorkload[wrWorkloadName]["agentNotAppliedOk"])

	assert.EqualValues(t, 2, overviewByWorkload[wrOtherName]["totalPods"])
	assert.EqualValues(t, 0, overviewByWorkload[wrOtherName]["totalAgentAppliedPods"])
	assert.EqualValues(t, 2, overviewByWorkload[wrOtherName]["totalAgentNotAppliedPods"])
	assert.Equal(t, false, overviewByWorkload[wrOtherName]["agentNotAppliedOk"],
		"a workload whose pods never got the agent must not report the uninjected count as ok")
}

// wrFieldChain builds a graphql.FieldContext chain, outermost ancestor first. gqlgen's
// WithFieldContext links each context to the one already on ctx, which is exactly how the
// generated marshalers build the chain the resolvers walk.
func wrFieldChain(ctx context.Context, results ...interface{}) context.Context {
	for _, result := range results {
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Result: result})
	}
	return ctx
}

func wrWorkloadResult() interface{} {
	id := wrWorkloadID()
	w := &model.K8sWorkload{ID: &id}
	return &w
}

func wrPodResult(podName string) interface{} {
	p := &model.K8sWorkloadPod{PodName: podName}
	return &p
}

// TestEveryNestedResolverRejectsAParentChainItCannotRead covers the defensive branches the
// real executor can never reach. They matter because the alternative to returning an error
// is a panic inside a gqlgen field goroutine, which takes down the whole request rather
// than one field. Each case asserts an error is returned, no value is produced, and the
// message names the parent chain so the failure is diagnosable from the API response.
func TestEveryNestedResolverRejectsAParentChainItCannotRead(t *testing.T) {
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrSimplePod(0, wrHealthyContainer(wrContainerName, wrReportingDistro)),
	)
	podName := wrPodName(wrWorkloadName, 0)
	notAWorkload := "definitely not a workload"

	processes := func(ctx context.Context) (interface{}, error) {
		return r.K8sWorkloadPodContainer().Processes(ctx, &model.K8sWorkloadPodContainer{ContainerName: wrContainerName})
	}
	overview := func(ctx context.Context) (interface{}, error) {
		return r.K8sWorkloadRollout().PodsManifestInjectionOverview(ctx, &model.K8sWorkloadRollout{})
	}
	expecting := func(ctx context.Context) (interface{}, error) {
		return r.K8sWorkloadTelemetryMetrics().ExpectingTelemetry(ctx, &model.K8sWorkloadTelemetryMetrics{})
	}

	cases := []struct {
		name    string
		resolve func(context.Context) (interface{}, error)
		// chain lists the ancestor Results, outermost first, with the resolver's own
		// field context appended last.
		chain []interface{}
	}{
		{"processes with no field context at all", processes, nil},
		{
			"processes with a chain too short to reach the pod",
			processes,
			[]interface{}{wrWorkloadResult(), nil, nil},
		},
		{
			"processes whose pod ancestor is not a pod",
			processes,
			[]interface{}{wrWorkloadResult(), nil, &notAWorkload, nil, nil, nil},
		},
		{
			"processes whose pod ancestor is a pod with no name",
			processes,
			[]interface{}{wrWorkloadResult(), nil, wrPodResult(""), nil, nil, nil},
		},
		{
			"processes with a chain too short to reach the workload",
			processes,
			[]interface{}{wrPodResult(podName), nil, nil, nil},
		},
		{
			"processes whose workload ancestor is not a workload",
			processes,
			[]interface{}{&notAWorkload, nil, wrPodResult(podName), nil, nil, nil},
		},
		{"overview with no field context at all", overview, nil},
		{
			"overview with a chain too short to reach the workload",
			overview,
			[]interface{}{nil, nil},
		},
		{
			"overview whose workload ancestor is not a workload",
			overview,
			[]interface{}{&notAWorkload, nil, nil},
		},
		{"expectingTelemetry with no field context at all", expecting, nil},
		{
			"expectingTelemetry with a chain too short to reach the workload",
			expecting,
			[]interface{}{nil, nil, nil},
		},
		{
			"expectingTelemetry whose workload ancestor is not a workload",
			expecting,
			[]interface{}{&notAWorkload, nil, nil, nil},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.resolve(wrFieldChain(ctx, tt.chain...))
			require.Error(t, err, "a chain the resolver cannot read must not produce a value")
			assert.Nil(t, got)
			assert.Contains(t, err.Error(), "parent",
				"the error must name the parent chain so the cause is diagnosable")
		})
	}
	require.Len(t, cases, 12, "each of the three resolvers has four unreadable-chain shapes")
}

// wrAncestorHops walks the parent chain from the given field context and reports, for each
// ancestor kind it recognises, how many .Parent hops away the nearest one is. It is the
// measurement the three resolvers hardcode as a literal chain of .Parent dereferences.
func wrAncestorHops(fc *graphql.FieldContext) map[string]int {
	hops := map[string]int{}
	for depth, cur := 1, fc.Parent; cur != nil; depth, cur = depth+1, cur.Parent {
		switch cur.Result.(type) {
		case **model.K8sWorkload:
			if _, seen := hops["workload"]; !seen {
				hops["workload"] = depth
			}
		case **model.K8sWorkloadPod:
			if _, seen := hops["pod"]; !seen {
				hops["pod"] = depth
			}
		}
	}
	return hops
}

// TestTheHardcodedAncestorDepthsMatchWhatGqlgenActuallyBuilds is the regression gate for
// the whole technique. It measures, from inside the real executor, how far up the parent
// chain each ancestor really sits, and pins those distances against the number of .Parent
// dereferences the resolver performs. Re-nesting a field in the schema — turning an object
// into a list, wrapping it in a new type — moves these numbers and fails here, instead of
// silently turning the resolver into its "parent is not a workload" error branch.
func TestTheHardcodedAncestorDepthsMatchWhatGqlgenActuallyBuilds(t *testing.T) {
	podName := wrPodName(wrWorkloadName, 0)
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
		wrSimplePod(0, wrHealthyContainer(wrContainerName, wrReportingDistro)),
		wrHealthyInstance(podName, wrContainerName, "5150"),
	)

	// gqlgen resolves sibling fields in their own goroutines, so the middleware writes
	// under a mutex.
	var mu sync.Mutex
	observed := map[string]map[string]int{}
	exec := executor.New(NewExecutableSchema(Config{Resolvers: r}))
	exec.AroundFields(func(ctx context.Context, next graphql.Resolver) (interface{}, error) {
		fc := graphql.GetFieldContext(ctx)
		switch fc.Field.Name {
		case "processes", "podsManifestInjectionOverview", "expectingTelemetry":
			hops := wrAncestorHops(fc)
			mu.Lock()
			observed[fc.Field.Name] = hops
			mu.Unlock()
		}
		return next(ctx)
	})

	traced := graphql.StartOperationTrace(ctx)
	opCtx, gqlErrs := exec.CreateOperationContext(traced,
		&graphql.RawParams{Query: wrParentContextQuery, Variables: wrSingleWorkloadVars()})
	require.Empty(t, gqlErrs)
	handler, traced := exec.DispatchOperation(traced, opCtx)
	require.Empty(t, handler(traced).Errors)

	mu.Lock()
	defer mu.Unlock()

	// processes climbs fc.Parent.Parent.Parent to the pod, then two more to the workload.
	assert.Equal(t, map[string]int{"pod": 3, "workload": 5}, observed["processes"])
	// podsManifestInjectionOverview climbs fc.Parent.Parent to the workload. It is two
	// rather than three only because `rollout` is a single object, not a list.
	assert.Equal(t, map[string]int{"workload": 2}, observed["podsManifestInjectionOverview"])
	// expectingTelemetry climbs fc.Parent.Parent.Parent to the workload, one more than
	// the overview because `telemetryMetrics` is a list.
	assert.Equal(t, map[string]int{"workload": 3}, observed["expectingTelemetry"])
	require.Len(t, observed, 3, "all three nested resolvers must have been exercised")
}

// TestTheNestedResolversAcceptTheExactChainDepthTheSchemaProduces is the other half of the
// rejection table: without it, a resolver that rejected every chain would still pass. The
// depths built here are the ones the test above measures the real executor producing.
func TestTheNestedResolversAcceptTheExactChainDepthTheSchemaProduces(t *testing.T) {
	podName := wrPodName(wrWorkloadName, 0)
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrSimplePod(0, wrHealthyContainer(wrContainerName, wrReportingDistro)),
		wrHealthyInstance(podName, wrContainerName, "5150"),
	)

	t.Run("processes reads the pod 3 hops up and the workload 5 hops up", func(t *testing.T) {
		// workload element -> pods field -> pod element -> containers field -> container
		// element -> processes
		chained := wrFieldChain(ctx, wrWorkloadResult(), nil, wrPodResult(podName), nil, nil, nil)
		got, err := r.K8sWorkloadPodContainer().Processes(chained,
			&model.K8sWorkloadPodContainer{ContainerName: wrContainerName})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, wrBool(true), got[0].Healthy)
	})

	t.Run("podsManifestInjectionOverview reads the workload 2 hops up", func(t *testing.T) {
		// workload element -> rollout field -> podsManifestInjectionOverview
		chained := wrFieldChain(ctx, wrWorkloadResult(), nil, nil)
		got, err := r.K8sWorkloadRollout().PodsManifestInjectionOverview(chained, &model.K8sWorkloadRollout{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, 1, got.TotalPods)
	})

	t.Run("expectingTelemetry reads the workload 3 hops up", func(t *testing.T) {
		// workload element -> telemetryMetrics field -> metrics element -> expectingTelemetry
		chained := wrFieldChain(ctx, wrWorkloadResult(), nil, nil, nil)
		got, err := r.K8sWorkloadTelemetryMetrics().ExpectingTelemetry(chained,
			&model.K8sWorkloadTelemetryMetrics{})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, wrBool(true), got.IsExpectingTelemetry)
	})
}

// TestPodsManifestInjectionOverviewShortCircuitsAnAlreadyPopulatedParent covers the guard
// that lets the eager list path pre-compute the overview: when it is already set, the
// resolver must return it without touching the parent chain at all. Driven with no field
// context, which is the state that would otherwise error.
func TestPodsManifestInjectionOverviewShortCircuitsAnAlreadyPopulatedParent(t *testing.T) {
	ctx, r := wrHarness(t,
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrReconciledIC(),
		wrSimplePod(0, wrHealthyContainer(wrContainerName, wrReportingDistro)),
	)

	precomputed := &model.K8sWorkloadPodsManifestInjectionOverview{TotalPods: 4242}
	got, err := r.K8sWorkloadRollout().PodsManifestInjectionOverview(ctx,
		&model.K8sWorkloadRollout{PodsManifestInjectionOverview: precomputed})
	require.NoError(t, err)
	assert.Same(t, precomputed, got,
		"a pre-computed overview must be returned as-is, not recomputed")
}
