package graph

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// wrPodsResolver is the pod list of the workload detail page: 80 lines that fan a single
// workload out into pods, containers, per-container kubernetes and odigos health, and two
// levels of aggregation.

func wrPodsFor(t *testing.T, objects ...client.Object) []*model.K8sWorkloadPod {
	t.Helper()
	ctx, r := wrHarness(t, objects...)
	id := wrWorkloadID()
	pods, err := r.K8sWorkload().Pods(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	return pods
}

func wrBaseObjects(extra ...client.Object) []client.Object {
	base := []client.Object{
		wrNamespaceObject(wrAppNamespace),
		wrDeployment(wrAppNamespace, wrWorkloadName, 1),
		wrSource(wrAppNamespace, wrWorkloadName, false, wrDataStream),
	}
	return append(base, extra...)
}

// TestPodsSortsPodsAndContainersByNameSoTheListIsStable pins the ordering the UI depends on
// instead of re-sorting. Both sorts run over data the fake client returns in an order the
// test deliberately does not match.
func TestPodsSortsPodsAndContainersByNameSoTheListIsStable(t *testing.T) {
	// seeded in descending order, and with the sidecar container declared first.
	pods := wrPodsFor(t, wrBaseObjects(
		wrReconciledIC(),
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 1), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers: []wrContainerFixture{
				wrHealthyContainer(wrSidecarName, wrReportingDistro),
				wrHealthyContainer(wrContainerName, wrReportingDistro),
			},
		}),
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers: []wrContainerFixture{
				wrHealthyContainer(wrSidecarName, wrReportingDistro),
				wrHealthyContainer(wrContainerName, wrReportingDistro),
			},
		}),
	)...)

	require.Len(t, pods, 2)
	assert.Equal(t, []string{wrPodName(wrWorkloadName, 0), wrPodName(wrWorkloadName, 1)},
		[]string{pods[0].PodName, pods[1].PodName})
	for _, pod := range pods {
		require.Len(t, pod.Containers, 2)
		assert.Equal(t, []string{wrContainerName, wrSidecarName},
			[]string{pod.Containers[0].ContainerName, pod.Containers[1].ContainerName},
			"containers of pod %q are out of order", pod.PodName)
	}
}

// TestPodsCopiesEveryComputedContainerFieldOntoItsOwnModelField guards the 11-field
// assignment block. Every field gets a distinct value in the fixture, so a copy/paste that
// wires two model fields to the same computed field is visible.
func TestPodsCopiesEveryComputedContainerFieldOntoItsOwnModelField(t *testing.T) {
	pods := wrPodsFor(t, wrBaseObjects(
		wrReconciledIC(),
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers: []wrContainerFixture{{
				name:          wrContainerName,
				distro:        wrReportingDistro,
				started:       true,
				ready:         false,
				waitingReason: "CrashLoopBackOff",
			}},
		}),
	)...)

	require.Len(t, pods, 1)
	pod := pods[0]
	assert.Equal(t, wrPodName(wrWorkloadName, 0), pod.PodName)
	assert.Equal(t, "wr-node", pod.NodeName)
	assert.Equal(t, "2026-01-02T03:04:05Z", pod.StartTime)
	assert.True(t, pod.AgentInjected)

	require.Len(t, pod.Containers, 1)
	c := pod.Containers[0]
	assert.Equal(t, wrContainerName, c.ContainerName)
	assert.Equal(t, wrStr(wrReportingDistro), c.OtelDistroName)
	assert.Equal(t, wrBool(true), c.Started)
	assert.Equal(t, wrBool(false), c.Ready, "ready must come from the container status, not from started")
	assert.Equal(t, wrBool(true), c.IsCrashLoop)
	assert.Equal(t, wrStr("CrashLoopBackOff"), c.WaitingReasonEnum)
	assert.Equal(t, wrStr("back-off restarting failed container"), c.WaitingMessage,
		"the waiting message must not be overwritten by the waiting reason")
	assert.Nil(t, c.RunningStartedTime, "a waiting container has no running start time")
	require.NotNil(t, c.RestartCount)
	assert.Equal(t, 0, *c.RestartCount)

	require.NotNil(t, c.K8sHealthStatus)
	assert.Equal(t, status.PodContainerHealthStatus, c.K8sHealthStatus.Name)
	assert.Equal(t, model.DesiredStateProgressFailure, c.K8sHealthStatus.Status)
	assert.Equal(t, wrStr(string(status.PodContainerK8sHealthReasonCrashLoopBackOff)), c.K8sHealthStatus.ReasonEnum)
}

// TestPodsSurfacesPodsOfAWorkloadThatHasNoInstrumentationConfig is the uninstrumented
// case, which is most of a fresh cluster. getContainerConfigByName is nil-safe, so the pod
// list must still render; only the odigos health status is absent.
func TestPodsSurfacesPodsOfAWorkloadThatHasNoInstrumentationConfig(t *testing.T) {
	pods := wrPodsFor(t, wrBaseObjects(
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			containers: []wrContainerFixture{wrHealthyContainer(wrContainerName, "")},
		}),
	)...)

	require.Len(t, pods, 1)
	require.Len(t, pods[0].Containers, 1)
	assert.Nil(t, pods[0].Containers[0].OdigosHealthStatus,
		"a container with no agent and no config reports no odigos health status")
	require.NotNil(t, pods[0].Containers[0].K8sHealthStatus,
		"kubernetes health is reported regardless of instrumentation")
	assert.Equal(t, model.DesiredStateProgressSuccess, pods[0].Containers[0].K8sHealthStatus.Status)
	assert.False(t, pods[0].AgentInjected)
}

// TestPodsOmitsTheOdigosHealthStatusOfNonAgentContainersFromThePodAggregate covers the
// conditional append: only containers that actually carry an agent contribute to the pod's
// odigos health. With an unconditional append a nil condition would reach the aggregation.
func TestPodsOmitsTheOdigosHealthStatusOfNonAgentContainersFromThePodAggregate(t *testing.T) {
	podName := wrPodName(wrWorkloadName, 0)
	pods := wrPodsFor(t, wrBaseObjects(
		wrReconciledIC(),
		wrPod(wrPodFixture{
			name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers: []wrContainerFixture{
				wrHealthyContainer(wrContainerName, wrReportingDistro),
				// no distro env var at all: this container has no agent.
				wrHealthyContainer(wrSidecarName, ""),
			},
		}),
		wrHealthyInstance(podName, wrContainerName, "31337"),
	)...)

	require.Len(t, pods, 1)
	require.Len(t, pods[0].Containers, 2)
	assert.NotNil(t, pods[0].Containers[0].OdigosHealthStatus, "the instrumented container reports odigos health")
	assert.Nil(t, pods[0].Containers[1].OdigosHealthStatus, "the plain container does not")

	require.NotNil(t, pods[0].OdigosHealthStatus)
	assert.Equal(t, model.DesiredStateProgressSuccess, pods[0].OdigosHealthStatus.Status)
	assert.Equal(t, wrStr(string(status.PodHealthOdigosStatusReasonHealthy)), pods[0].OdigosHealthStatus.ReasonEnum)
}

// TestPodsReportsAnInstrumentedContainerWithNoProcessesAsWaiting pins the distro allow
// list at the resolver level: a container whose distro is expected to report
// InstrumentationInstances but has none is waiting, while one whose distro never reports
// them is healthy on the strength of the injected agent alone. Without the second half the
// allow list could be neutralised to "always expecting" unnoticed.
func TestPodsReportsAnInstrumentedContainerWithNoProcessesAsWaiting(t *testing.T) {
	for name, tt := range map[string]struct {
		distro     string
		wantStatus model.DesiredStateProgress
		wantReason status.PodHealthOdigosStatusReason
	}{
		"a distro that reports processes": {
			distro:     wrReportingDistro,
			wantStatus: model.DesiredStateProgressWaiting,
			wantReason: status.PodHealthOdigosStatusReasonNoInstrumentedProcesses,
		},
		"a distro that never reports processes": {
			distro:     wrSilentDistro,
			wantStatus: model.DesiredStateProgressSuccess,
			wantReason: status.PodHealthOdigosStatusReasonHealthy,
		},
	} {
		t.Run(name, func(t *testing.T) {
			pods := wrPodsFor(t, wrBaseObjects(
				wrReconciledIC(),
				wrPod(wrPodFixture{
					name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
					agentsMetaHash: wrAgentsMetaHash,
					containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, tt.distro)},
				}),
				// deliberately no InstrumentationInstance.
			)...)

			require.Len(t, pods, 1)
			require.Len(t, pods[0].Containers, 1)
			got := pods[0].Containers[0].OdigosHealthStatus
			require.NotNil(t, got)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, wrStr(string(tt.wantReason)), got.ReasonEnum)
		})
	}
}

// TestPodsReportsOdigosHealthForANoRestartContainerWithNoDistroEnvVar is the only fixture
// in which the per-container odigos config changes the answer. A container with no
// ODIGOS_DISTRO_NAME env var normally reports no odigos health at all, but when its agent
// can be enabled without a pod restart the config says so and the container must be
// reported. Without this case, never looking the config up at all is invisible.
func TestPodsReportsOdigosHealthForANoRestartContainerWithNoDistroEnvVar(t *testing.T) {
	noRestartConfig := func(ic *v1alpha1.InstrumentationConfig) {
		ic.Spec.PodManifestInjectionOptional = true
		ic.Spec.Containers = []v1alpha1.ContainerAgentConfig{{
			ContainerName:                wrContainerName,
			AgentEnabled:                 true,
			PodManifestInjectionOptional: true,
		}}
	}
	// the pod manifest carries no distro env var, which is the whole point.
	podWithoutDistro := wrPod(wrPodFixture{
		name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
		agentsMetaHash: wrAgentsMetaHash,
		containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, "")},
	})

	t.Run("the config marks the agent as injectable without a restart", func(t *testing.T) {
		pods := wrPodsFor(t, wrBaseObjects(wrReconciledIC(noRestartConfig), podWithoutDistro)...)
		require.Len(t, pods, 1)
		require.Len(t, pods[0].Containers, 1)
		got := pods[0].Containers[0].OdigosHealthStatus
		require.NotNil(t, got, "a no-restart container must report odigos health")
		assert.Equal(t, model.DesiredStateProgressSuccess, got.Status)
		assert.Equal(t, wrStr(string(status.PodHealthOdigosStatusReasonHealthy)), got.ReasonEnum)
	})

	t.Run("the same pod without that config reports nothing", func(t *testing.T) {
		pods := wrPodsFor(t, wrBaseObjects(wrReconciledIC(), podWithoutDistro)...)
		require.Len(t, pods, 1)
		require.Len(t, pods[0].Containers, 1)
		assert.Nil(t, pods[0].Containers[0].OdigosHealthStatus,
			"without the no-restart config the container has no agent to report on")
	})
}

// TestPodsLooksUpInstrumentationInstancesPerPodAndContainer is the same three-string
// identity check the parent-context suite applies to Processes, but for the PodContainerId
// that Pods assembles itself.
func TestPodsLooksUpInstrumentationInstancesPerPodAndContainer(t *testing.T) {
	podA, podB := wrPodName(wrWorkloadName, 0), wrPodName(wrWorkloadName, 1)
	unhealthyInstance := wrInstance(wrInstanceFixture{
		name:          "unhealthy-sidecar-of-pod-b",
		namespace:     wrAppNamespace,
		workloadName:  wrWorkloadName,
		podName:       podB,
		containerName: wrSidecarName,
		healthy:       wrBool(false),
		message:       "agent failed to start",
	})

	twoContainers := []wrContainerFixture{
		wrHealthyContainer(wrContainerName, wrReportingDistro),
		wrHealthyContainer(wrSidecarName, wrReportingDistro),
	}
	pods := wrPodsFor(t, wrBaseObjects(
		wrReconciledIC(func(ic *v1alpha1.InstrumentationConfig) {
			ic.Spec.Containers = append(ic.Spec.Containers, v1alpha1.ContainerAgentConfig{
				ContainerName: wrSidecarName, AgentEnabled: true, OtelDistroName: wrReportingDistro,
			})
		}),
		wrPod(wrPodFixture{name: podA, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash, containers: twoContainers}),
		wrPod(wrPodFixture{name: podB, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash, containers: twoContainers}),
		wrHealthyInstance(podA, wrContainerName, "1001"),
		wrHealthyInstance(podA, wrSidecarName, "1002"),
		wrHealthyInstance(podB, wrContainerName, "2001"),
		unhealthyInstance,
	)...)

	require.Len(t, pods, 2)
	// exactly one of the four (pod, container) pairs is unhealthy; every other pair must
	// be unaffected by it.
	failures := 0
	for _, pod := range pods {
		require.Len(t, pod.Containers, 2)
		for _, c := range pod.Containers {
			require.NotNil(t, c.OdigosHealthStatus, "pod %q container %q", pod.PodName, c.ContainerName)
			if c.OdigosHealthStatus.Status == model.DesiredStateProgressFailure {
				failures++
				assert.Equal(t, podB, pod.PodName)
				assert.Equal(t, wrSidecarName, c.ContainerName)
				assert.Contains(t, c.OdigosHealthStatus.Message, "agent failed to start")
			} else {
				assert.Equal(t, model.DesiredStateProgressSuccess, c.OdigosHealthStatus.Status,
					"pod %q container %q must be healthy", pod.PodName, c.ContainerName)
			}
		}
	}
	require.Equal(t, 1, failures, "exactly one container may be reported unhealthy")

	// and the failure must propagate to that pod's aggregate only.
	byName := map[string]*model.K8sWorkloadPod{pods[0].PodName: pods[0], pods[1].PodName: pods[1]}
	assert.Equal(t, model.DesiredStateProgressSuccess, byName[podA].OdigosHealthStatus.Status)
	assert.Equal(t, model.DesiredStateProgressFailure, byName[podB].OdigosHealthStatus.Status)
}

// TestPodsHealthStatusRewritesTheAggregateMessageForEveryReason covers the four-branch
// message rewrite. The messages differ only as strings, so the table drives each branch
// and then asserts the four are mutually distinct: a copy/paste between branches would
// otherwise leave two reasons describing the same thing.
func TestPodsHealthStatusRewritesTheAggregateMessageForEveryReason(t *testing.T) {
	// the three unhealthy container shapes, each of which must win the aggregation
	// against a healthy sibling.
	cases := []struct {
		name       string
		container  wrContainerFixture
		wantReason status.PodContainerK8sHealthReason
		wantStatus model.DesiredStateProgress
	}{
		{
			name:       "all containers healthy",
			container:  wrHealthyContainer(wrSidecarName, wrReportingDistro),
			wantReason: status.PodContainerK8sHealthReasonHealthy,
			wantStatus: model.DesiredStateProgressSuccess,
		},
		{
			name:       "a container that has not started",
			container:  wrContainerFixture{name: wrSidecarName, distro: wrReportingDistro, started: false, ready: false},
			wantReason: status.PodContainerK8sHealthReasonNotStarted,
			wantStatus: model.DesiredStateProgressWaiting,
		},
		{
			name:       "a container that started but is not ready",
			container:  wrContainerFixture{name: wrSidecarName, distro: wrReportingDistro, started: true, ready: false},
			wantReason: status.PodContainerK8sHealthReasonNotReady,
			wantStatus: model.DesiredStateProgressWaiting,
		},
		{
			name: "a container in crash loop back off",
			container: wrContainerFixture{name: wrSidecarName, distro: wrReportingDistro,
				started: true, ready: false, waitingReason: "CrashLoopBackOff"},
			wantReason: status.PodContainerK8sHealthReasonCrashLoopBackOff,
			wantStatus: model.DesiredStateProgressFailure,
		},
	}

	messages := map[status.PodContainerK8sHealthReason]string{}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx, r := wrHarness(t, wrBaseObjects(
				wrReconciledIC(),
				wrPod(wrPodFixture{
					name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
					agentsMetaHash: wrAgentsMetaHash,
					containers: []wrContainerFixture{
						wrHealthyContainer(wrContainerName, wrReportingDistro),
						tt.container,
					},
				}),
			)...)

			id := wrWorkloadID()
			got, err := r.K8sWorkload().PodsHealthStatus(ctx, &model.K8sWorkload{ID: &id})
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, wrStr(string(tt.wantReason)), got.ReasonEnum)
			// every branch must produce workload-level phrasing. Without this, a branch
			// keyed on the wrong reason falls through and the container's own message —
			// which talks about one container — is served as the workload summary.
			assert.Contains(t, got.Message, "pods",
				"the workload summary must talk about the workload's pods")
			messages[tt.wantReason] = got.Message
		})
	}

	require.Len(t, messages, len(cases), "every reason must have produced a message")
	seen := map[string]status.PodContainerK8sHealthReason{}
	for reason, message := range messages {
		if other, collision := seen[message]; collision {
			t.Fatalf("reasons %q and %q render the identical message %q", other, reason, message)
		}
		seen[message] = reason
	}
}

// TestTheWorkloadLevelPodHealthMessagesDifferFromThePodLevelOnes pins the split between
// two parallel message tables keyed by the same reason enum: status.getPodStatusMessageFromReason
// describes one pod, while PodsHealthStatus rewrites the same reason to describe all the
// pods of a workload. Copying a phrase from one table to the other makes the workload
// summary claim something about a single pod.
func TestTheWorkloadLevelPodHealthMessagesDifferFromThePodLevelOnes(t *testing.T) {
	podName := wrPodName(wrWorkloadName, 0)
	objects := wrBaseObjects(
		wrReconciledIC(),
		wrPod(wrPodFixture{
			name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
		}),
		wrHealthyInstance(podName, wrContainerName, "4242"),
	)

	ctx, r := wrHarness(t, objects...)
	id := wrWorkloadID()

	pods, err := r.K8sWorkload().Pods(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.Len(t, pods, 1)
	require.NotNil(t, pods[0].K8sHealthStatus)

	workloadLevel, err := r.K8sWorkload().PodsHealthStatus(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, workloadLevel)

	assert.Equal(t, pods[0].K8sHealthStatus.ReasonEnum, workloadLevel.ReasonEnum,
		"both layers must agree on the reason")
	assert.NotEqual(t, pods[0].K8sHealthStatus.Message, workloadLevel.Message,
		"the workload summary must not reuse the single-pod phrasing")
	assert.Contains(t, workloadLevel.Message, "all pods")
}

// TestPodsHealthStatusLabelsItsAggregateAsAContainerConditionWhenPodsExist characterises a
// current inconsistency rather than asserting a desired behaviour.
// PodsHealthStatus returns the aggregate object that AggregateConditionsBySeverity picked,
// which is one of the per-container conditions, and only overwrites its Message. So the
// workload-level podsHealthStatus field reports Name "PodCotainerHealthK8s" when the
// workload has pods, but "PodHealthK8s" when it has none — two different names from one
// non-null field, and neither matches the pod-level condition built by
// status.CalculatePodHealthK8sStatus. Pinned so that changing either branch is a
// deliberate decision.
func TestPodsHealthStatusLabelsItsAggregateAsAContainerConditionWhenPodsExist(t *testing.T) {
	id := wrWorkloadID()

	withPods, r := wrHarness(t, wrBaseObjects(
		wrReconciledIC(),
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
		}),
	)...)
	populated, err := r.K8sWorkload().PodsHealthStatus(withPods, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, populated)

	empty, emptyResolver := wrHarness(t, wrBaseObjects(wrReconciledIC())...)
	unpopulated, err := emptyResolver.K8sWorkload().PodsHealthStatus(empty, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, unpopulated)

	assert.Equal(t, status.PodContainerHealthStatus, populated.Name)
	assert.Equal(t, status.PodHealthStatus, unpopulated.Name)
	assert.NotEqual(t, populated.Name, unpopulated.Name,
		"the two branches of podsHealthStatus disagree about the condition name")
}

// TestPodsHealthStatusReportsAnErrorWhenThereAreNoContainersToAggregate covers the nil
// aggregate branch. A workload with no running pods has nothing to aggregate, and the
// resolver must say so rather than return a nil condition into a non-null schema field.
func TestPodsHealthStatusReportsAnErrorWhenThereAreNoContainersToAggregate(t *testing.T) {
	ctx, r := wrHarness(t, wrBaseObjects(wrReconciledIC())...)
	id := wrWorkloadID()

	got, err := r.K8sWorkload().PodsHealthStatus(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, got, "podsHealthStatus is a non-null schema field")
	assert.Equal(t, status.PodHealthStatus, got.Name)
	assert.Equal(t, model.DesiredStateProgressError, got.Status)
	assert.Equal(t, wrStr(string(status.PodContainerK8sHealthReasonUnknown)), got.ReasonEnum)
	assert.NotEmpty(t, got.Message)
}

// TestPodsOdigosHealthStatusDistinguishesNoPodsFromNoAgent is the guard that the empty-pod
// case has its own outcome. Both shapes produce "nothing to report", but the UI tells the
// user to start a pod in one case and to roll out the workload in the other.
func TestPodsOdigosHealthStatusDistinguishesNoPodsFromNoAgent(t *testing.T) {
	id := wrWorkloadID()

	t.Run("no pods at all", func(t *testing.T) {
		ctx, r := wrHarness(t, wrBaseObjects(wrReconciledIC())...)
		got, err := r.K8sWorkload().PodsOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, status.PodHealthOdigosStatus, got.Name)
		assert.Equal(t, model.DesiredStateProgressUnknown, got.Status)
		assert.Equal(t, wrStr(string(status.PodHealthOdigosStatusReasonNoPods)), got.ReasonEnum)
	})

	t.Run("a pod that never got the agent injected", func(t *testing.T) {
		ctx, r := wrHarness(t, wrBaseObjects(
			wrReconciledIC(),
			// no agents-meta-hash label: the webhook never mutated this pod.
			wrPod(wrPodFixture{
				name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
				containers: []wrContainerFixture{wrHealthyContainer(wrContainerName, "")},
			}),
		)...)
		got, err := r.K8sWorkload().PodsOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, wrStr(string(status.PodHealthOdigosStatusReasonNotInjected)), got.ReasonEnum)
		assert.NotEqual(t, model.DesiredStateProgressSuccess, got.Status)
	})

	t.Run("a fully instrumented pod", func(t *testing.T) {
		podName := wrPodName(wrWorkloadName, 0)
		ctx, r := wrHarness(t, wrBaseObjects(
			wrReconciledIC(),
			wrPod(wrPodFixture{
				name: podName, workloadName: wrWorkloadName, namespace: wrAppNamespace,
				agentsMetaHash: wrAgentsMetaHash,
				containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
			}),
			wrHealthyInstance(podName, wrContainerName, "1234"),
		)...)
		got, err := r.K8sWorkload().PodsOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, model.DesiredStateProgressSuccess, got.Status)
		assert.Equal(t, wrStr(string(status.PodHealthOdigosStatusReasonHealthy)), got.ReasonEnum)
	})
}

// TestPodsOdigosHealthStatusAggregatesAcrossPodsNotJustWithinOne covers the outer append.
// With two pods where only the second is unhealthy, dropping the cross-pod aggregation
// would report the first pod's verdict for the workload.
func TestPodsOdigosHealthStatusAggregatesAcrossPodsNotJustWithinOne(t *testing.T) {
	podA, podB := wrPodName(wrWorkloadName, 0), wrPodName(wrWorkloadName, 1)
	containers := []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)}

	ctx, r := wrHarness(t, wrBaseObjects(
		wrReconciledIC(),
		wrPod(wrPodFixture{name: podA, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash, containers: containers}),
		wrPod(wrPodFixture{name: podB, workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash, containers: containers}),
		wrHealthyInstance(podA, wrContainerName, "1001"),
		wrInstance(wrInstanceFixture{
			name: "unhealthy-of-pod-b", namespace: wrAppNamespace, workloadName: wrWorkloadName,
			podName: podB, containerName: wrContainerName, healthy: wrBool(false),
			message: "the agent crashed",
		}),
	)...)

	id := wrWorkloadID()
	got, err := r.K8sWorkload().PodsOdigosHealthStatus(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, model.DesiredStateProgressFailure, got.Status,
		"one unhealthy pod out of two must make the workload aggregate unhealthy")
	assert.Equal(t, wrStr(string(status.PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy)), got.ReasonEnum)
	assert.Contains(t, got.Message, "the agent crashed")
}

// TestPodsAgentInjectionStatusShortCircuitsAPrePopulatedValue covers both halves of the
// eager-population guard: the pre-computed condition must be returned by identity, and an
// unpopulated workload must be computed from the cluster.
func TestPodsAgentInjectionStatusShortCircuitsAPrePopulatedValue(t *testing.T) {
	ctx, r := wrHarness(t, wrBaseObjects(
		wrReconciledIC(),
		wrPod(wrPodFixture{
			name: wrPodName(wrWorkloadName, 0), workloadName: wrWorkloadName, namespace: wrAppNamespace,
			agentsMetaHash: wrAgentsMetaHash,
			containers:     []wrContainerFixture{wrHealthyContainer(wrContainerName, wrReportingDistro)},
		}),
	)...)
	id := wrWorkloadID()

	precomputed := &model.DesiredConditionStatus{Name: "precomputed", Status: model.DesiredStateProgressIrrelevant}
	got, err := r.K8sWorkload().PodsAgentInjectionStatus(ctx,
		&model.K8sWorkload{ID: &id, PodsAgentInjectionStatus: precomputed})
	require.NoError(t, err)
	assert.Same(t, precomputed, got, "a pre-computed condition must be returned as-is")

	computed, err := r.K8sWorkload().PodsAgentInjectionStatus(ctx, &model.K8sWorkload{ID: &id})
	require.NoError(t, err)
	require.NotNil(t, computed)
	assert.NotSame(t, precomputed, computed)
	assert.Equal(t, status.AgentInjectedStatus, computed.Name)
	assert.Equal(t, model.DesiredStateProgressSuccess, computed.Status)
}
