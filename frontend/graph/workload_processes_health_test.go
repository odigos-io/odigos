package graph

import (
	"context"
	"errors"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// The health aggregation is a chain of eight mutually-exclusive guards evaluated in a fixed order.
// Whichever one fires first is what the workload page reports, and a wrong answer surfaces as a
// plausible-looking status rather than an error, so every outcome needs its own fixture.

func phRequireStatus(t *testing.T, got *model.DesiredConditionStatus, wantProgress model.DesiredStateProgress, wantReason status.ProcessesHealthStatusReason, wantMessage string) {
	t.Helper()
	require.NotNil(t, got)
	assert.Equal(t, status.ProcessesHealthStatusName, got.Name)
	assert.Equal(t, wantProgress, got.Status)
	require.NotNil(t, got.ReasonEnum)
	assert.Equal(t, string(wantReason), *got.ReasonEnum)
	assert.Equal(t, wantMessage, got.Message)
}

func phAggregate(t *testing.T, fixture wuFixture, optional map[string]struct{}) *model.DesiredConditionStatus {
	t.Helper()
	ctx := wuLoaderContext(t, fixture)
	id := wuWorkloadID()
	got, err := aggregateProcessesHealthForWorkload(ctx, &id, optional)
	require.NoError(t, err)
	return got
}

func phHealthyComponent(name string) v1alpha1.InstrumentationLibraryStatus {
	return v1alpha1.InstrumentationLibraryStatus{
		Name:    name,
		Type:    v1alpha1.InstrumentationLibraryTypeInstrumentation,
		Healthy: wuBool(true),
	}
}

func phUnhealthyComponent(name string, message string) v1alpha1.InstrumentationLibraryStatus {
	return v1alpha1.InstrumentationLibraryStatus{
		Name:    name,
		Type:    v1alpha1.InstrumentationLibraryTypeInstrumentation,
		Healthy: wuBool(false),
		Message: message,
	}
}

// phInstrumentedPod is the baseline: one pod, one ready container running a distro that reports
// instrumentation instances. Every guard-chain test varies exactly one thing from here.
func phInstrumentedPod() wuPod {
	return wuPod{name: "checkout-abc", containers: []wuContainer{{name: "app", distro: wuReportingDistro, ready: true}}}
}

func phInstanceFor(healthy *bool, message string, components ...v1alpha1.InstrumentationLibraryStatus) wuInstance {
	return wuInstance{podName: "checkout-abc", containerName: "app", healthy: healthy, message: message, components: components}
}

func TestAggregateProcessesHealth_AllComponentsHealthyIsReportedAsSuccess(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(true), "", phHealthyComponent("net/http"), phHealthyComponent("database/sql"))},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressSuccess, status.ProcessesHealthStatusReasonAllHealthy,
		"All 2 agents in instrumented processes are healthy")
}

func TestAggregateProcessesHealth_AnUnhealthyInstanceIsAFailureCarryingItsMessage(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(false), "failed to load the agent")},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"Found 1 processes with unhealthy agent: failed to load the agent")
}

func TestAggregateProcessesHealth_AnUnhealthyInstanceWithNoMessageReportsOnlyTheCount(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(false), "")},
	}, nil)

	// the count must still be reported; only the ": <details>" suffix is dropped.
	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"Found 1 processes with unhealthy agent")
}

func TestAggregateProcessesHealth_EveryUnhealthyInstanceMessageIsJoinedInOrder(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{
			phInstanceFor(wuBool(false), "first process died"),
			phInstanceFor(wuBool(false), "second process died"),
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"Found 2 processes with unhealthy agent: first process died; second process died")
}

func TestAggregateProcessesHealth_AHealthyInstanceWithAnUnhealthyComponentIsAFailure(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(true), "",
			phHealthyComponent("net/http"),
			phUnhealthyComponent("database/sql", "driver not supported"),
		)},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"unhealthy instrumentation libraries in 1 processes: driver not supported")
}

func TestAggregateProcessesHealth_AnUnhealthyComponentWithNoMessageReportsOnlyTheCount(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(true), "", phUnhealthyComponent("database/sql", ""))},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"unhealthy instrumentation libraries in 1 processes")
}

// The scan stops at the first unhealthy component of an instance, so the reported number is the
// number of PROCESSES carrying at least one unhealthy library — not the number of libraries —
// which is what the message claims.
func TestAggregateProcessesHealth_SeveralUnhealthyComponentsInOneProcessCountOnce(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(true), "",
			phUnhealthyComponent("database/sql", "driver not supported"),
			phUnhealthyComponent("net/http", "handler panicked"),
		)},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"unhealthy instrumentation libraries in 1 processes: driver not supported")
}

func TestAggregateProcessesHealth_UnhealthyComponentsInSeveralProcessesAreCountedAndJoined(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{
			phInstanceFor(wuBool(true), "", phUnhealthyComponent("database/sql", "driver not supported")),
			phInstanceFor(wuBool(true), "", phUnhealthyComponent("net/http", "handler panicked")),
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"unhealthy instrumentation libraries in 2 processes: driver not supported; handler panicked")
}

// A component whose Healthy is nil is "not reported", which must not be read as unhealthy.
func TestAggregateProcessesHealth_AComponentWithNoHealthReportedIsNotTreatedAsUnhealthy(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(true), "",
			v1alpha1.InstrumentationLibraryStatus{Name: "net/http", Type: v1alpha1.InstrumentationLibraryTypeInstrumentation},
			phHealthyComponent("database/sql"),
		)},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressSuccess, status.ProcessesHealthStatusReasonAllHealthy,
		"All 2 agents in instrumented processes are healthy")
}

// An unhealthy INSTANCE outranks an unhealthy COMPONENT: the two report the same reason enum but a
// different message, and the process-level failure is the actionable one.
func TestAggregateProcessesHealth_AnUnhealthyProcessOutranksAnUnhealthyComponent(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{
			phInstanceFor(wuBool(true), "", phUnhealthyComponent("database/sql", "driver not supported")),
			phInstanceFor(wuBool(false), "process crashed"),
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"Found 1 processes with unhealthy agent: process crashed")
}

// The unhealthy checks run before every "not ready yet" check, so a failure is never masked by a
// second pod that is still starting.
func TestAggregateProcessesHealth_AnUnhealthyProcessIsReportedEvenWhileAnotherPodIsStarting(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{
			phInstrumentedPod(),
			{name: "checkout-def", containers: []wuContainer{{name: "app", distro: wuReportingDistro, ready: true}}},
		},
		instances: []wuInstance{
			phInstanceFor(wuBool(false), "process crashed"),
			{podName: "checkout-def", containerName: "app", healthy: nil},
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressFailure, status.ProcessesHealthStatusReasonOdigosUnhealthyInSomeProcesses,
		"Found 1 processes with unhealthy agent: process crashed")
}

func TestAggregateProcessesHealth_NoAgentOnAnyContainerIsIrrelevant(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", ready: true}}}},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressIrrelevant, status.ProcessesHealthStatusReasonNoAgentInjected,
		"none of the running pods is instrumented with odigos agent")
}

func TestAggregateProcessesHealth_AWorkloadWithNoPodsAtAllIsIrrelevant(t *testing.T) {
	got := phAggregate(t, wuFixture{}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressIrrelevant, status.ProcessesHealthStatusReasonNoAgentInjected,
		"none of the running pods is instrumented with odigos agent")
}

// A distro outside the reporting allow list never produces instrumentation instances, so the
// absence of instances is expected rather than a problem — "unsupported", not "waiting".
func TestAggregateProcessesHealth_ADistroThatDoesNotReportInstancesIsUnsupported(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", distro: wuSilentDistro, ready: true}}}},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressIrrelevant, status.ProcessesHealthStatusReasonUnsupported,
		"agents used in this workload does not support health status reporting")
}

// "unsupported" must not swallow a workload that also runs a reporting distro elsewhere: the
// reporting container decides, and here it has no instance yet.
func TestAggregateProcessesHealth_AReportingContainerAlongsideASilentOneStillExpectsInstances(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{
			{name: "sidecar", distro: wuSilentDistro, ready: true},
			{name: "app", distro: wuReportingDistro, ready: true},
		}}},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonNoProcesses,
		"agent not yet started in all instrumented containers")
}

func TestAggregateProcessesHealth_AnInstrumentedContainerThatIsNotReadyIsWaiting(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", distro: wuReportingDistro, ready: false}}}},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonContainersNotReady,
		"agent not yet started in any instrumented containers")
}

// "not ready" outranks "missing instances": a container that has not reported ready is not
// expected to have produced an instance yet, so reporting NoProcesses would be misleading.
func TestAggregateProcessesHealth_ANotReadyContainerOutranksAReadyOneMissingItsInstance(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{
			{name: "checkout-abc", containers: []wuContainer{{name: "app", distro: wuReportingDistro, ready: false}}},
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonContainersNotReady,
		"agent not yet started in any instrumented containers")
}

func TestAggregateProcessesHealth_AReadyContainerWithNoInstanceIsWaitingOnProcesses(t *testing.T) {
	got := phAggregate(t, wuFixture{pods: []wuPod{phInstrumentedPod()}}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonNoProcesses,
		"agent not yet started in all instrumented containers")
}

// A single container still missing its instance holds the whole workload at NoProcesses, even
// when a sibling container is already fully healthy.
func TestAggregateProcessesHealth_OneContainerMissingItsInstanceOutranksAHealthySibling(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{
			{name: "app", distro: wuReportingDistro, ready: true},
			{name: "worker", distro: wuReportingDistro, ready: true},
		}}},
		instances: []wuInstance{phInstanceFor(wuBool(true), "", phHealthyComponent("net/http"))},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonNoProcesses,
		"agent not yet started in all instrumented containers")
}

// Healthy==nil means the agent has not reported yet.
func TestAggregateProcessesHealth_AnInstanceWithNoHealthYetIsStarting(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(nil, "")},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonStarting,
		"Found 1 processes with starting agent")
}

// "starting" outranks "all healthy" so the UI does not claim success while a process is still
// coming up.
func TestAggregateProcessesHealth_AStartingProcessOutranksAHealthyOne(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{
			phInstanceFor(wuBool(true), "", phHealthyComponent("net/http")),
			phInstanceFor(nil, ""),
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonStarting,
		"Found 1 processes with starting agent")
}

// A healthy instance that reports no components at all leaves every counter at zero, so the whole
// chain falls through and the workload page renders no processes-health condition. Characterised
// rather than endorsed — see the PR description.
func TestAggregateProcessesHealth_AHealthyInstanceWithNoComponentsProducesNoStatusAtAll(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{phInstrumentedPod()},
		instances: []wuInstance{phInstanceFor(wuBool(true), "")},
	}, nil)

	assert.Nil(t, got)
}

// The reported number is a count of instrumentation LIBRARIES, not of processes, even though the
// message says "agents in instrumented processes". Pinned so the two stay in sync deliberately.
func TestAggregateProcessesHealth_TheHealthyCountIsPerComponentNotPerProcess(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{
			phInstanceFor(wuBool(true), "", phHealthyComponent("net/http"), phHealthyComponent("database/sql")),
			phInstanceFor(wuBool(true), "", phHealthyComponent("net/http")),
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressSuccess, status.ProcessesHealthStatusReasonAllHealthy,
		"All 3 agents in instrumented processes are healthy")
}

// Containers that odigos injects without touching the pod manifest carry no ODIGOS_DISTRO_NAME env
// var, so without the optional-injection set they look uninstrumented and the workload is reported
// as having no agent at all.
func TestAggregateProcessesHealth_AnOptionalPodManifestInjectionContainerCountsAsInstrumented(t *testing.T) {
	fixture := wuFixture{
		pods:      []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", ready: true}}}},
		instances: []wuInstance{phInstanceFor(wuBool(true), "", phHealthyComponent("net/http"))},
	}

	withoutOptional := phAggregate(t, fixture, nil)
	phRequireStatus(t, withoutOptional, model.DesiredStateProgressIrrelevant, status.ProcessesHealthStatusReasonNoAgentInjected,
		"none of the running pods is instrumented with odigos agent")

	withOptional := phAggregate(t, fixture, map[string]struct{}{"app": {}})
	phRequireStatus(t, withOptional, model.DesiredStateProgressSuccess, status.ProcessesHealthStatusReasonAllHealthy,
		"All 1 agents in instrumented processes are healthy")
}

// The optional set is keyed by container name, so naming a different container must not promote
// the uninstrumented one.
func TestAggregateProcessesHealth_TheOptionalInjectionSetIsMatchedByContainerName(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", ready: true}}}},
		instances: []wuInstance{phInstanceFor(wuBool(true), "", phHealthyComponent("net/http"))},
	}, map[string]struct{}{"some-other-container": {}})

	phRequireStatus(t, got, model.DesiredStateProgressIrrelevant, status.ProcessesHealthStatusReasonNoAgentInjected,
		"none of the running pods is instrumented with odigos agent")
}

// An optional-injection container bypasses the distro allow-list check entirely: instances are
// always expected for it, so a missing one is NoProcesses rather than Unsupported.
func TestAggregateProcessesHealth_AnOptionalInjectionContainerAlwaysExpectsInstances(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", distro: wuSilentDistro, ready: true}}}},
	}, map[string]struct{}{"app": {}})

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonNoProcesses,
		"agent not yet started in all instrumented containers")
}

// An optional-injection container that is not ready is still subject to the readiness gate.
func TestAggregateProcessesHealth_AnOptionalInjectionContainerThatIsNotReadyIsWaiting(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", ready: false}}}},
	}, map[string]struct{}{"app": {}})

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonContainersNotReady,
		"agent not yet started in any instrumented containers")
}

// Instances are looked up per (namespace, pod, container); an instance belonging to a different
// pod of the same workload must not satisfy this container.
func TestAggregateProcessesHealth_AnInstanceFromAnotherPodDoesNotSatisfyThisContainer(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		instances: []wuInstance{
			{podName: "checkout-zzz", containerName: "app", healthy: wuBool(true), components: []v1alpha1.InstrumentationLibraryStatus{phHealthyComponent("net/http")}},
		},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonNoProcesses,
		"agent not yet started in all instrumented containers")
}

// Same for an instance belonging to a different container of the same pod.
func TestAggregateProcessesHealth_AnInstanceFromAnotherContainerDoesNotSatisfyThisOne(t *testing.T) {
	got := phAggregate(t, wuFixture{
		pods:      []wuPod{phInstrumentedPod()},
		instances: []wuInstance{{podName: "checkout-abc", containerName: "sidecar", healthy: wuBool(true)}},
	}, nil)

	phRequireStatus(t, got, model.DesiredStateProgressWaiting, status.ProcessesHealthStatusReasonNoProcesses,
		"agent not yet started in all instrumented containers")
}

// A read failure must surface as an error rather than as one of the "nothing found" outcomes,
// which would render a green/irrelevant workload while the cluster is unreadable.

func TestAggregateProcessesHealth_AFailureListingPodsIsReportedAsAnError(t *testing.T) {
	ctx := wuLoaderContext(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		cacheFuncs: interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, isPods := list.(*corev1.PodList); isPods {
					return errors.New("pod informer cache is not synced")
				}
				return c.List(ctx, list, opts...)
			},
		},
	})

	id := wuWorkloadID()
	got, err := aggregateProcessesHealthForWorkload(ctx, &id, nil)

	assert.Nil(t, got)
	require.Error(t, err)
	assert.ErrorContains(t, err, "pod informer cache is not synced")
}

func TestAggregateProcessesHealth_AFailureListingInstrumentationInstancesIsReportedAsAnError(t *testing.T) {
	ctx := wuLoaderContext(t, wuFixture{
		pods: []wuPod{phInstrumentedPod()},
		cacheFuncs: interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, isInstances := list.(*v1alpha1.InstrumentationInstanceList); isInstances {
					return errors.New("instrumentation instance informer cache is not synced")
				}
				return c.List(ctx, list, opts...)
			},
		},
	})

	id := wuWorkloadID()
	got, err := aggregateProcessesHealthForWorkload(ctx, &id, nil)

	assert.Nil(t, got)
	require.Error(t, err)
	assert.ErrorContains(t, err, "instrumentation instance informer cache is not synced")
}

// Every outcome must carry a reason the UI can switch on, and no two outcomes may share a
// (progress, reason, message) triple — otherwise the page cannot tell them apart.
func TestAggregateProcessesHealth_EveryOutcomeIsDistinguishable(t *testing.T) {
	outcomes := map[string]*model.DesiredConditionStatus{
		"unhealthy process": phAggregate(t, wuFixture{
			pods:      []wuPod{phInstrumentedPod()},
			instances: []wuInstance{phInstanceFor(wuBool(false), "")},
		}, nil),
		"unhealthy component": phAggregate(t, wuFixture{
			pods:      []wuPod{phInstrumentedPod()},
			instances: []wuInstance{phInstanceFor(wuBool(true), "", phUnhealthyComponent("database/sql", ""))},
		}, nil),
		"no agent": phAggregate(t, wuFixture{
			pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", ready: true}}}},
		}, nil),
		"unsupported": phAggregate(t, wuFixture{
			pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", distro: wuSilentDistro, ready: true}}}},
		}, nil),
		"containers not ready": phAggregate(t, wuFixture{
			pods: []wuPod{{name: "checkout-abc", containers: []wuContainer{{name: "app", distro: wuReportingDistro, ready: false}}}},
		}, nil),
		"no processes": phAggregate(t, wuFixture{pods: []wuPod{phInstrumentedPod()}}, nil),
		"starting": phAggregate(t, wuFixture{
			pods:      []wuPod{phInstrumentedPod()},
			instances: []wuInstance{phInstanceFor(nil, "")},
		}, nil),
		"all healthy": phAggregate(t, wuFixture{
			pods:      []wuPod{phInstrumentedPod()},
			instances: []wuInstance{phInstanceFor(wuBool(true), "", phHealthyComponent("net/http"))},
		}, nil),
	}

	seen := make(map[string]string, len(outcomes))
	for name, got := range outcomes {
		require.NotNil(t, got, name)
		require.NotNil(t, got.ReasonEnum, name)
		assert.Equal(t, status.ProcessesHealthStatusName, got.Name, name)
		assert.NotEmpty(t, got.Message, name)

		key := string(got.Status) + "|" + *got.ReasonEnum + "|" + got.Message
		if previous, collides := seen[key]; collides {
			t.Errorf("%q and %q are indistinguishable to the UI: %s", previous, name, key)
		}
		seen[key] = name
	}
	assert.Len(t, seen, len(outcomes))
}
