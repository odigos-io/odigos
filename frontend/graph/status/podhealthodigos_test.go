package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func odigosHealthContainer(otelDistroName string, expectingInstances bool) *computed.ComputedPodContainer {
	container := &computed.ComputedPodContainer{
		ContainerName:                     "app",
		ExpectingInstrumentationInstances: expectingInstances,
		IsReady:                           true,
	}
	if otelDistroName != "" {
		container.OtelDistroName = &otelDistroName
	}
	return container
}

func unhealthyComponent(message string) v1alpha1.InstrumentationLibraryStatus {
	unhealthy := false
	return v1alpha1.InstrumentationLibraryStatus{
		Name:    "http",
		Type:    v1alpha1.InstrumentationLibraryTypeInstrumentation,
		Healthy: &unhealthy,
		Message: message,
	}
}

// A container with no agent must produce no odigos condition at all: the
// callers append only non-nil conditions, so returning one here would drag
// uninstrumented sidecars into the pod's odigos health.
func TestCalculatePodContainerHealthOdigosStatusNoAgentInContainer(t *testing.T) {
	tests := []struct {
		name            string
		containerConfig *v1alpha1.ContainerAgentConfig
	}{
		{name: "no container config", containerConfig: nil},
		{
			name:            "container config without optional pod manifest injection",
			containerConfig: &v1alpha1.ContainerAgentConfig{ContainerName: "app", PodManifestInjectionOptional: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePodContainerHealthOdigosStatus(odigosHealthContainer("", false), tt.containerConfig, nil)
			if got != nil {
				t.Fatalf("expected nil, got %+v", got)
			}
		})
	}
}

// A "no restart required" agent is enabled without appearing in the pod
// manifest, so the container has no distro env var but is still instrumented
// and must be reported on.
func TestCalculatePodContainerHealthOdigosStatusOptionalInjectionWithoutDistroEnvVar(t *testing.T) {
	got := CalculatePodContainerHealthOdigosStatus(
		odigosHealthContainer("", false),
		&v1alpha1.ContainerAgentConfig{ContainerName: "app", PodManifestInjectionOptional: true},
		nil,
	)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressSuccess {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressSuccess, got.Status)
	}
	if reasonOf(t, got) != string(PodHealthOdigosStatusReasonHealthy) {
		t.Fatalf("expected reason %q, got %q", PodHealthOdigosStatusReasonHealthy, reasonOf(t, got))
	}
	if got.Message != "odigos agent is running in the container" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

func TestCalculatePodContainerHealthOdigosStatus(t *testing.T) {
	healthy, unhealthy := true, false

	tests := []struct {
		name        string
		container   *computed.ComputedPodContainer
		instances   []*v1alpha1.InstrumentationInstance
		wantStatus  model.DesiredStateProgress
		wantReason  PodHealthOdigosStatusReason
		wantMessage string
	}{
		{
			name:        "agent injected, no instances expected and none reported",
			container:   odigosHealthContainer("nodejs-community", false),
			instances:   nil,
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  PodHealthOdigosStatusReasonHealthy,
			wantMessage: "odigos agent is running in the container",
		},
		{
			// The distro reports instrumentation instances but none arrived,
			// which is how a failed agent load surfaces.
			name:        "instances expected but none reported",
			container:   odigosHealthContainer("nodejs-community", true),
			instances:   nil,
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  PodHealthOdigosStatusReasonNoInstrumentedProcesses,
			wantMessage: "no instrumented processes found in the container",
		},
		{
			name:        "healthy instances",
			container:   odigosHealthContainer("nodejs-community", true),
			instances:   []*v1alpha1.InstrumentationInstance{instrumentationInstance(&healthy, "running")},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  PodHealthOdigosStatusReasonHealthy,
			wantMessage: "all instrumented processes are reported as healthy",
		},
		{
			// An instance that has not reported health yet is not a failure.
			name:        "instance health not reported yet",
			container:   odigosHealthContainer("nodejs-community", true),
			instances:   []*v1alpha1.InstrumentationInstance{instrumentationInstance(nil, "starting")},
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  PodHealthOdigosStatusReasonHealthy,
			wantMessage: "all instrumented processes are reported as healthy",
		},
		{
			name:        "unhealthy instance with a message",
			container:   odigosHealthContainer("nodejs-community", true),
			instances:   []*v1alpha1.InstrumentationInstance{instrumentationInstance(&unhealthy, "failed to load")},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy,
			wantMessage: "instrumented process in the container is unhealthy: failed to load",
		},
		{
			name:        "unhealthy instance without a message",
			container:   odigosHealthContainer("nodejs-community", true),
			instances:   []*v1alpha1.InstrumentationInstance{instrumentationInstance(&unhealthy, "")},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy,
			wantMessage: "instrumented process in the container is unhealthy",
		},
		{
			// The process is healthy overall but one instrumentation library
			// inside it is not, which must not be reported as healthy.
			name:      "healthy instance with an unhealthy library",
			container: odigosHealthContainer("nodejs-community", true),
			instances: []*v1alpha1.InstrumentationInstance{{
				Status: v1alpha1.InstrumentationInstanceStatus{
					Healthy:    &healthy,
					Components: []v1alpha1.InstrumentationLibraryStatus{unhealthyComponent("unsupported version")},
				},
			}},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy,
			wantMessage: "unhealthy instrumentation library in instrumented process: unsupported version",
		},
		{
			name:      "unhealthy library without a message",
			container: odigosHealthContainer("nodejs-community", true),
			instances: []*v1alpha1.InstrumentationInstance{{
				Status: v1alpha1.InstrumentationInstanceStatus{
					Healthy:    &healthy,
					Components: []v1alpha1.InstrumentationLibraryStatus{unhealthyComponent("")},
				},
			}},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy,
			wantMessage: "unhealthy instrumentation library in instrumented process",
		},
		{
			// An unhealthy process is reported even when instances are present,
			// i.e. the health scan runs before the "no instances" check.
			name:      "unhealthy instance among healthy ones",
			container: odigosHealthContainer("nodejs-community", true),
			instances: []*v1alpha1.InstrumentationInstance{
				instrumentationInstance(&healthy, "running"),
				instrumentationInstance(&unhealthy, "crashed"),
			},
			wantStatus:  model.DesiredStateProgressFailure,
			wantReason:  PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy,
			wantMessage: "instrumented process in the container is unhealthy: crashed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePodContainerHealthOdigosStatus(tt.container, nil, tt.instances)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != PodContainerHealthOdigosStatus {
				t.Fatalf("expected name %q, got %q", PodContainerHealthOdigosStatus, got.Name)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, got.Status)
			}
			if reasonOf(t, got) != string(tt.wantReason) {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, reasonOf(t, got))
			}
			if got.Message != tt.wantMessage {
				t.Fatalf("expected message %q, got %q", tt.wantMessage, got.Message)
			}
		})
	}
}

// Whether the agent made it into the pod is checked before the containers'
// odigos health, because a pod that never got the agent has nothing to report
// about instrumented processes.
func TestCalculatePodHealthOdigosStatusAgentInjectionNotSuccessful(t *testing.T) {
	tests := []struct {
		name          string
		agentInjected bool
		injectedState model.DesiredStateProgress
		wantReason    PodHealthOdigosStatusReason
	}{
		{
			name:          "agent missing from a pod that should have it",
			agentInjected: false,
			injectedState: model.DesiredStateProgressNotice,
			wantReason:    PodHealthOdigosStatusReasonNotInjected,
		},
		{
			name:          "agent present in a pod that should not have it",
			agentInjected: true,
			injectedState: model.DesiredStateProgressWaiting,
			wantReason:    PodHealthOdigosStatusReasonInjectedUninstrumentedSource,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &computed.CachedPod{
				PodName:       "pod-1",
				AgentInjected: tt.agentInjected,
				AgentInjectedStatus: &model.DesiredConditionStatus{
					Name:    PodAgentInjectionStatus,
					Status:  tt.injectedState,
					Message: "agent injection message",
				},
			}

			// A healthy container condition must not hide the injection problem.
			got := CalculatePodHealthOdigosStatus(pod, []*model.DesiredConditionStatus{
				createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonHealthy, "healthy", model.DesiredStateProgressSuccess),
			})

			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != PodHealthOdigosStatus {
				t.Fatalf("expected name %q, got %q", PodHealthOdigosStatus, got.Name)
			}
			if reasonOf(t, got) != string(tt.wantReason) {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, reasonOf(t, got))
			}
			// severity and message are carried over from the injection status.
			if got.Status != tt.injectedState {
				t.Fatalf("expected status %q, got %q", tt.injectedState, got.Status)
			}
			if got.Message != "agent injection message" {
				t.Fatalf("unexpected message %q", got.Message)
			}
		})
	}
}

func TestCalculatePodHealthOdigosStatusAgentInjected(t *testing.T) {
	injectedPod := func() *computed.CachedPod {
		return &computed.CachedPod{
			PodName:       "pod-1",
			AgentInjected: true,
			AgentInjectedStatus: &model.DesiredConditionStatus{
				Name:   PodAgentInjectionStatus,
				Status: model.DesiredStateProgressSuccess,
			},
		}
	}

	t.Run("no container reported an odigos condition", func(t *testing.T) {
		// Every container in the pod is uninstrumented, so the pod has no
		// odigos health to report and the caller skips it.
		if got := CalculatePodHealthOdigosStatus(injectedPod(), nil); got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("unhealthy container is surfaced as is", func(t *testing.T) {
		unhealthyContainer := createPodContainerHealthOdigosStatus(
			PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy,
			"instrumented process in the container is unhealthy: crashed",
			model.DesiredStateProgressFailure,
		)

		got := CalculatePodHealthOdigosStatus(injectedPod(), []*model.DesiredConditionStatus{
			createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonHealthy, "healthy", model.DesiredStateProgressSuccess),
			unhealthyContainer,
		})

		// The container condition itself is returned, keeping the container
		// level message that names the failure.
		if got != unhealthyContainer {
			t.Fatalf("expected the container condition, got %+v", got)
		}
	})

	t.Run("all containers healthy", func(t *testing.T) {
		got := CalculatePodHealthOdigosStatus(injectedPod(), []*model.DesiredConditionStatus{
			createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonHealthy, "healthy", model.DesiredStateProgressSuccess),
			createPodContainerHealthOdigosStatus(PodHealthOdigosStatusReasonHealthy, "healthy", model.DesiredStateProgressSuccess),
		})

		if got == nil {
			t.Fatal("expected a status, got nil")
		}
		if got.Name != PodHealthOdigosStatus {
			t.Fatalf("expected name %q, got %q", PodHealthOdigosStatus, got.Name)
		}
		if got.Status != model.DesiredStateProgressSuccess {
			t.Fatalf("expected status %q, got %q", model.DesiredStateProgressSuccess, got.Status)
		}
		if reasonOf(t, got) != string(PodHealthOdigosStatusReasonHealthy) {
			t.Fatalf("expected reason %q, got %q", PodHealthOdigosStatusReasonHealthy, reasonOf(t, got))
		}
		if got.Message != "odigos instrumentation in pod is healthy" {
			t.Fatalf("unexpected message %q", got.Message)
		}
	})
}

// The workload level condition is built from the most severe pod condition, but
// the per-pod message ("this pod...") would be misleading for a workload, so
// some reasons are rewritten and the rest pass through untouched.
func TestMostSeverPodStatusToAggregated(t *testing.T) {
	t.Run("rewritten reasons", func(t *testing.T) {
		tests := []struct {
			reason      PodHealthOdigosStatusReason
			status      model.DesiredStateProgress
			wantStatus  model.DesiredStateProgress
			wantMessage string
		}{
			{
				reason:      PodHealthOdigosStatusReasonHealthy,
				status:      model.DesiredStateProgressSuccess,
				wantStatus:  model.DesiredStateProgressSuccess,
				wantMessage: "odigos is healthy in all pods",
			},
			{
				reason:      PodHealthOdigosStatusReasonNotInjected,
				status:      model.DesiredStateProgressNotice,
				wantStatus:  model.DesiredStateProgressNotice,
				wantMessage: "not all pods are running with odigos agent",
			},
			{
				reason:      PodHealthOdigosStatusReasonInjectedUninstrumentedSource,
				status:      model.DesiredStateProgressWaiting,
				wantStatus:  model.DesiredStateProgressWaiting,
				wantMessage: "not all pods are running without odigos agent",
			},
		}

		for _, tt := range tests {
			t.Run(string(tt.reason), func(t *testing.T) {
				got := MostSeverPodStatusToAggregated(createPodHealthOdigosStatus(tt.reason, "per pod message", tt.status))
				if got == nil {
					t.Fatal("expected a status, got nil")
				}
				if reasonOf(t, got) != string(tt.reason) {
					t.Fatalf("expected reason %q, got %q", tt.reason, reasonOf(t, got))
				}
				if got.Status != tt.wantStatus {
					t.Fatalf("expected status %q, got %q", tt.wantStatus, got.Status)
				}
				if got.Message != tt.wantMessage {
					t.Fatalf("expected message %q, got %q", tt.wantMessage, got.Message)
				}
			})
		}
	})

	t.Run("reasons passed through untouched", func(t *testing.T) {
		reasons := []PodHealthOdigosStatusReason{
			PodHealthOdigosStatusReasonInstrumentatedProcessUnhealthy,
			PodHealthOdigosStatusReasonNoInstrumentedProcesses,
			PodHealthOdigosStatusReasonNoPods,
			PodHealthOdigosStatusReason("SomeReasonAddedLater"),
		}

		for _, reason := range reasons {
			t.Run(string(reason), func(t *testing.T) {
				// These messages already name the concrete failure, so they are
				// more useful than a generic workload level rewrite.
				in := createPodHealthOdigosStatus(reason, "instrumented process in the container is unhealthy: crashed", model.DesiredStateProgressFailure)
				if got := MostSeverPodStatusToAggregated(in); got != in {
					t.Fatalf("expected the input condition, got %+v", got)
				}
			})
		}
	})

	t.Run("no pod condition to aggregate", func(t *testing.T) {
		if got := MostSeverPodStatusToAggregated(nil); got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
		if got := MostSeverPodStatusToAggregated(&model.DesiredConditionStatus{Name: PodHealthOdigosStatus}); got != nil {
			t.Fatalf("expected nil for a condition without a reason, got %+v", got)
		}
	})
}
