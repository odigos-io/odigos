package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func telemetryContainer(name, otelDistroName string, isReady bool) computed.ComputedPodContainer {
	container := computed.ComputedPodContainer{ContainerName: name, IsReady: isReady}
	if otelDistroName != "" {
		container.OtelDistroName = &otelDistroName
	}
	return container
}

func telemetryPod(agentInjected bool, containers ...computed.ComputedPodContainer) computed.CachedPod {
	return computed.CachedPod{
		PodNamespace:  "default",
		PodName:       "checkout-abc123",
		AgentInjected: agentInjected,
		Containers:    containers,
	}
}

func telemetryIC(agentInjectionEnabled bool, optionalInjectionContainers ...string) *v1alpha1.InstrumentationConfig {
	ic := &v1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = agentInjectionEnabled
	for _, containerName := range optionalInjectionContainers {
		ic.Spec.Containers = append(ic.Spec.Containers, v1alpha1.ContainerAgentConfig{
			ContainerName:                containerName,
			PodManifestInjectionOptional: true,
		})
	}
	return ic
}

// IsExpectingTelemetry gates the "no data received" warning in the UI. If it is
// true too early, every workload that is still rolling out is reported as
// broken; if it stays false, a genuinely silent agent is never surfaced.
func TestCalculateExpectingTelemetryStatus(t *testing.T) {
	dataSent := 4096
	noDataSent := 0

	tests := []struct {
		name          string
		ic            *v1alpha1.InstrumentationConfig
		pods          []computed.CachedPod
		totalDataSent *int
		wantExpecting bool
		wantStatus    model.DesiredStateProgress
		wantReason    ExpectingTelemetryReason
		wantMessage   string
	}{
		{
			name:          "workload not marked for instrumentation",
			ic:            nil,
			pods:          []computed.CachedPod{telemetryPod(true, telemetryContainer("app", "nodejs-community", true))},
			wantExpecting: false,
			wantStatus:    model.DesiredStateProgressIrrelevant,
			wantReason:    ExpectingTelemetryReasonWorkloadNotMarkedForInstrumentation,
			wantMessage:   "workload is not marked for instrumentation",
		},
		{
			name:          "agent injection not enabled",
			ic:            telemetryIC(false),
			pods:          []computed.CachedPod{telemetryPod(true, telemetryContainer("app", "nodejs-community", true))},
			wantExpecting: false,
			wantStatus:    model.DesiredStateProgressIrrelevant,
			wantReason:    ExpectingTelemetryReasonAgentNotEnabledForInjection,
			wantMessage:   "agent injection is not enabled for this workload, no telemetry is expected",
		},
		{
			name:          "no pods",
			ic:            telemetryIC(true),
			pods:          nil,
			wantExpecting: false,
			wantStatus:    model.DesiredStateProgressPending,
			wantReason:    ExpectingTelemetryReasonNoRunningPod,
			wantMessage:   "no running pods found for this workload, no telemetry is expected",
		},
		{
			name:          "pod without the agent injected",
			ic:            telemetryIC(true),
			pods:          []computed.CachedPod{telemetryPod(false, telemetryContainer("app", "nodejs-community", true))},
			wantExpecting: false,
			wantStatus:    model.DesiredStateProgressIrrelevant,
			wantReason:    ExpectingTelemetryReasonAgentNotInjected,
			wantMessage:   "no instrumented container in running state yet, telemetry is not yet expected",
		},
		{
			name:          "injected pod with no instrumented container",
			ic:            telemetryIC(true),
			pods:          []computed.CachedPod{telemetryPod(true, telemetryContainer("app", "", true))},
			wantExpecting: false,
			wantStatus:    model.DesiredStateProgressIrrelevant,
			wantReason:    ExpectingTelemetryReasonAgentNotInjected,
			wantMessage:   "no instrumented container in running state yet, telemetry is not yet expected",
		},
		{
			name:          "instrumented container not ready yet",
			ic:            telemetryIC(true),
			pods:          []computed.CachedPod{telemetryPod(true, telemetryContainer("app", "nodejs-community", false))},
			wantExpecting: false,
			wantStatus:    model.DesiredStateProgressWaiting,
			wantReason:    ExpectingTelemetryReasonInstrumentedContainersNotReady,
			wantMessage:   "instrumented containers are not in ready state, telemetry is not yet expected",
		},
		{
			name:          "ready instrumented container, no data yet",
			ic:            telemetryIC(true),
			pods:          []computed.CachedPod{telemetryPod(true, telemetryContainer("app", "nodejs-community", true))},
			totalDataSent: nil,
			wantExpecting: true,
			wantStatus:    model.DesiredStateProgressWaiting,
			wantReason:    ExpectingTelemetryReasonAgentWaitingForTelemetry,
			wantMessage:   "source is instrumented and healthy, waiting for telemetry to be collected",
		},
		{
			name:          "ready instrumented container, zero bytes sent",
			ic:            telemetryIC(true),
			pods:          []computed.CachedPod{telemetryPod(true, telemetryContainer("app", "nodejs-community", true))},
			totalDataSent: &noDataSent,
			wantExpecting: true,
			wantStatus:    model.DesiredStateProgressWaiting,
			wantReason:    ExpectingTelemetryReasonAgentWaitingForTelemetry,
			wantMessage:   "source is instrumented and healthy, waiting for telemetry to be collected",
		},
		{
			name:          "ready instrumented container with data sent",
			ic:            telemetryIC(true),
			pods:          []computed.CachedPod{telemetryPod(true, telemetryContainer("app", "nodejs-community", true))},
			totalDataSent: &dataSent,
			wantExpecting: true,
			wantStatus:    model.DesiredStateProgressSuccess,
			wantReason:    ExpectingTelemetryReasonAgentInjectedAndDataSent,
			wantMessage:   "workload is reporting telemetry data",
		},
		{
			// One ready instrumented container anywhere in the workload is
			// enough, even if another pod is still starting.
			name: "one of two pods is ready",
			ic:   telemetryIC(true),
			pods: []computed.CachedPod{
				telemetryPod(true, telemetryContainer("app", "nodejs-community", false)),
				telemetryPod(true, telemetryContainer("app", "nodejs-community", true)),
			},
			totalDataSent: &dataSent,
			wantExpecting: true,
			wantStatus:    model.DesiredStateProgressSuccess,
			wantReason:    ExpectingTelemetryReasonAgentInjectedAndDataSent,
			wantMessage:   "workload is reporting telemetry data",
		},
		{
			// A "no restart required" agent leaves no distro env var in the pod
			// manifest, so the container is recognized by name from the config.
			name:          "optional injection container without a distro env var",
			ic:            telemetryIC(true, "app"),
			pods:          []computed.CachedPod{telemetryPod(false, telemetryContainer("app", "", true))},
			totalDataSent: &dataSent,
			wantExpecting: true,
			wantStatus:    model.DesiredStateProgressSuccess,
			wantReason:    ExpectingTelemetryReasonAgentInjectedAndDataSent,
			wantMessage:   "workload is reporting telemetry data",
		},
		{
			name:          "optional injection container not ready",
			ic:            telemetryIC(true, "app"),
			pods:          []computed.CachedPod{telemetryPod(false, telemetryContainer("app", "", false))},
			wantExpecting: false,
			wantStatus:    model.DesiredStateProgressWaiting,
			wantReason:    ExpectingTelemetryReasonInstrumentedContainersNotReady,
			wantMessage:   "instrumented containers are not in ready state, telemetry is not yet expected",
		},
		{
			// Once any container uses optional injection, pods without the
			// agent label are no longer skipped, so a normally injected
			// container in an unlabelled pod is examined too.
			name:          "uninjected pod is inspected when the workload has an optional injection container",
			ic:            telemetryIC(true, "sidecar"),
			pods:          []computed.CachedPod{telemetryPod(false, telemetryContainer("app", "nodejs-community", true))},
			totalDataSent: &dataSent,
			wantExpecting: true,
			wantStatus:    model.DesiredStateProgressSuccess,
			wantReason:    ExpectingTelemetryReasonAgentInjectedAndDataSent,
			wantMessage:   "workload is reporting telemetry data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateExpectingTelemetryStatus(tt.ic, tt.pods, tt.totalDataSent)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.IsExpectingTelemetry == nil {
				t.Fatal("expected IsExpectingTelemetry to be set")
			}
			if *got.IsExpectingTelemetry != tt.wantExpecting {
				t.Fatalf("expected IsExpectingTelemetry %t, got %t", tt.wantExpecting, *got.IsExpectingTelemetry)
			}
			if got.TelemetryObservedStatus == nil {
				t.Fatal("expected an observed status, got nil")
			}
			if got.TelemetryObservedStatus.Name != ExpectingTelemetryStatus {
				t.Fatalf("expected name %q, got %q", ExpectingTelemetryStatus, got.TelemetryObservedStatus.Name)
			}
			if got.TelemetryObservedStatus.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, got.TelemetryObservedStatus.Status)
			}
			if reasonOf(t, got.TelemetryObservedStatus) != string(tt.wantReason) {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, reasonOf(t, got.TelemetryObservedStatus))
			}
			if got.TelemetryObservedStatus.Message != tt.wantMessage {
				t.Fatalf("expected message %q, got %q", tt.wantMessage, got.TelemetryObservedStatus.Message)
			}
		})
	}
}

// The distro name comes from an env var in the pod manifest, so it can be
// present but empty. That is not an instrumented container.
func TestCalculateExpectingTelemetryStatusEmptyDistroName(t *testing.T) {
	emptyDistroName := ""
	pod := telemetryPod(true, computed.ComputedPodContainer{
		ContainerName:  "app",
		OtelDistroName: &emptyDistroName,
		IsReady:        true,
	})

	dataSent := 4096
	got := CalculateExpectingTelemetryStatus(telemetryIC(true), []computed.CachedPod{pod}, &dataSent)

	if *got.IsExpectingTelemetry {
		t.Fatal("expected not to be expecting telemetry for a container with an empty distro name")
	}
	if reasonOf(t, got.TelemetryObservedStatus) != string(ExpectingTelemetryReasonAgentNotInjected) {
		t.Fatalf("expected reason %q, got %q", ExpectingTelemetryReasonAgentNotInjected, reasonOf(t, got.TelemetryObservedStatus))
	}
}

// An instrumented container in one pod must not be credited to a pod that has
// no agent at all, which is what happens if the per-pod skip is dropped.
func TestCalculateExpectingTelemetryStatusUninjectedPodIsSkipped(t *testing.T) {
	got := CalculateExpectingTelemetryStatus(telemetryIC(true), []computed.CachedPod{
		telemetryPod(false, telemetryContainer("app", "nodejs-community", true)),
		telemetryPod(false, telemetryContainer("app", "nodejs-community", true)),
	}, nil)

	if *got.IsExpectingTelemetry {
		t.Fatal("expected not to be expecting telemetry when no pod has the agent injected")
	}
	if reasonOf(t, got.TelemetryObservedStatus) != string(ExpectingTelemetryReasonAgentNotInjected) {
		t.Fatalf("expected reason %q, got %q", ExpectingTelemetryReasonAgentNotInjected, reasonOf(t, got.TelemetryObservedStatus))
	}
}

// Only containers named as optional in the config bypass the distro env var
// check; a different container in the same pod still needs its distro.
func TestCalculateExpectingTelemetryStatusOptionalInjectionIsPerContainer(t *testing.T) {
	got := CalculateExpectingTelemetryStatus(telemetryIC(true, "sidecar"), []computed.CachedPod{
		telemetryPod(true, telemetryContainer("app", "", true)),
	}, nil)

	if *got.IsExpectingTelemetry {
		t.Fatal("expected not to be expecting telemetry for an uninstrumented container")
	}
	if reasonOf(t, got.TelemetryObservedStatus) != string(ExpectingTelemetryReasonAgentNotInjected) {
		t.Fatalf("expected reason %q, got %q", ExpectingTelemetryReasonAgentNotInjected, reasonOf(t, got.TelemetryObservedStatus))
	}
}
