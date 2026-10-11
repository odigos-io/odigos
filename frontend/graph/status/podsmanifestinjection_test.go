package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	podsManifestInjectionStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCalculatePodsManifestInjectionStatusIncludesActionItems(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{
		Status: v1alpha1.InstrumentationConfigStatus{
			Conditions: []metav1.Condition{
				{
					Type:    podsManifestInjectionStatus.PodsManifestInjectionType,
					Reason:  string(podsManifestInjectionStatus.PodsManifestInjectionReasonWaitingInRolloutQueue_Enabled),
					Message: "waiting for rollout",
				},
			},
		},
	}

	got := CalculatePodsManifestInjectionStatus(ic, nil)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if len(got.ActionItems) != 1 {
		t.Fatalf("expected one action item, got %d", len(got.ActionItems))
	}
	if got.ActionItems[0].Type != model.DesiredConditionActionItemTypeRolloutWorkload {
		t.Fatalf("expected action type %q, got %q", model.DesiredConditionActionItemTypeRolloutWorkload, got.ActionItems[0].Type)
	}
	if got.ActionItems[0].ButtonText == "" {
		t.Fatal("expected action item button text")
	}
}

func TestCalculatePodsManifestInjectionStatusNotYetReconciled(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{
		Status: v1alpha1.InstrumentationConfigStatus{},
	}

	got := CalculatePodsManifestInjectionStatus(ic, nil)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressWaiting {
		t.Fatalf("expected waiting, got %q", got.Status)
	}
	if got.ReasonEnum == nil || *got.ReasonEnum != podsManifestInjectionStatus.PodsManifestInjectionNotYetReconciled.Title {
		t.Fatalf("expected reason title %q, got %v", podsManifestInjectionStatus.PodsManifestInjectionNotYetReconciled.Title, got.ReasonEnum)
	}
}

func TestCalculatePodsManifestInjectionStatusUnmarkedNoPods(t *testing.T) {
	got := CalculatePodsManifestInjectionStatus(nil, nil)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressIrrelevant {
		t.Fatalf("expected irrelevant, got %q", got.Status)
	}
	if got.ReasonEnum == nil || *got.ReasonEnum != podsManifestInjectionStatus.PodsManifestInjectionNoPods.Title {
		t.Fatalf("expected reason title %q, got %v", podsManifestInjectionStatus.PodsManifestInjectionNoPods.Title, got.ReasonEnum)
	}
}

func TestCalculatePodsManifestInjectionStatusUnmarkedWithoutAgentLabel(t *testing.T) {
	got := CalculatePodsManifestInjectionStatus(nil, []computed.CachedPod{
		{PodName: "pod-1", AgentInjected: false},
	})
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressSuccess {
		t.Fatalf("expected success, got %q", got.Status)
	}
	if got.ReasonEnum == nil || *got.ReasonEnum != podsManifestInjectionStatus.PodsManifestInjectionPodsAppliedSuccessfully_Disabled.Title {
		t.Fatalf("expected reason title %q, got %v", podsManifestInjectionStatus.PodsManifestInjectionPodsAppliedSuccessfully_Disabled.Title, got.ReasonEnum)
	}
}

func TestCalculatePodsManifestInjectionStatusUnmarkedWithAgentLabel(t *testing.T) {
	got := CalculatePodsManifestInjectionStatus(nil, []computed.CachedPod{
		{PodName: "pod-1", AgentInjected: true, AgentsMetaHash: "abc"},
	})
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressNotice {
		t.Fatalf("expected notice, got %q", got.Status)
	}
	if got.ReasonEnum == nil || *got.ReasonEnum != podsManifestInjectionStatus.PodsManifestInjectionUnmarkedFromOdigos_Disabled.Title {
		t.Fatalf("expected reason title %q, got %v", podsManifestInjectionStatus.PodsManifestInjectionUnmarkedFromOdigos_Disabled.Title, got.ReasonEnum)
	}
	if len(got.ActionItems) != 1 {
		t.Fatalf("expected one action item, got %d", len(got.ActionItems))
	}
}

// A reason written by a newer instrumentor than this UI is surfaced with an
// unknown severity rather than mapped to a wrong one.
func TestCalculatePodsManifestInjectionStatusUnknownReason(t *testing.T) {
	ic := &v1alpha1.InstrumentationConfig{
		Status: v1alpha1.InstrumentationConfigStatus{
			Conditions: []metav1.Condition{
				{
					Type:    podsManifestInjectionStatus.PodsManifestInjectionType,
					Reason:  "SomeReasonAddedLater",
					Message: "a message from the future",
				},
			},
		},
	}

	got := CalculatePodsManifestInjectionStatus(ic, nil)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Name != podsManifestInjectionStatus.PodsManifestInjectionType {
		t.Fatalf("expected name %q, got %q", podsManifestInjectionStatus.PodsManifestInjectionType, got.Name)
	}
	if got.Status != model.DesiredStateProgressUnknown {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressUnknown, got.Status)
	}
	if got.ReasonEnum != nil {
		t.Fatalf("expected no reason, got %q", *got.ReasonEnum)
	}
	if got.Message != "a message from the future" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

func overviewIC(agentInjectionEnabled bool, agentsMetaHash string, injectionOptional bool) *v1alpha1.InstrumentationConfig {
	ic := &v1alpha1.InstrumentationConfig{}
	ic.Spec.AgentInjectionEnabled = agentInjectionEnabled
	ic.Spec.AgentsMetaHash = agentsMetaHash
	ic.Spec.PodManifestInjectionOptional = injectionOptional
	return ic
}

func overviewPod(podName, agentsMetaHash string) computed.CachedPod {
	return computed.CachedPod{
		PodName:        podName,
		AgentsMetaHash: agentsMetaHash,
		AgentInjected:  agentsMetaHash != "",
	}
}

// The overview drives the per-pod breakdown in the UI. The "ok" flags decide
// whether each bucket is drawn as a problem, and they are inverted for a
// workload that should not be instrumented: there, pods that DO carry the agent
// are the problem.
func TestCalculatePodsManifestInjectionOverview(t *testing.T) {
	tests := []struct {
		name             string
		ic               *v1alpha1.InstrumentationConfig
		pods             []computed.CachedPod
		wantTotal        int
		wantNotApplied   int
		wantNotAppliedOk bool
		wantApplied      int
		wantAppliedOk    bool
		wantOutOfDate    int
		wantOutOfDateOk  bool
	}{
		{
			name:             "workload not marked for instrumentation with no pods",
			ic:               nil,
			pods:             nil,
			wantNotAppliedOk: true,
			wantAppliedOk:    true,
			wantOutOfDateOk:  true,
		},
		{
			// Leftover instrumented pods on an unmarked workload are counted as
			// applied, and applied is what is NOT ok here.
			name:             "workload not marked for instrumentation with an instrumented pod",
			ic:               nil,
			pods:             []computed.CachedPod{overviewPod("pod-1", "hash-current"), overviewPod("pod-2", "")},
			wantTotal:        2,
			wantNotApplied:   1,
			wantNotAppliedOk: true,
			wantApplied:      1,
			wantAppliedOk:    false,
			wantOutOfDateOk:  true,
		},
		{
			name:             "agent enabled and every pod is up to date",
			ic:               overviewIC(true, "hash-current", false),
			pods:             []computed.CachedPod{overviewPod("pod-1", "hash-current"), overviewPod("pod-2", "hash-current")},
			wantTotal:        2,
			wantNotAppliedOk: true,
			wantApplied:      2,
			wantAppliedOk:    true,
			wantOutOfDateOk:  true,
		},
		{
			name:             "agent enabled and a pod carries an old hash",
			ic:               overviewIC(true, "hash-current", false),
			pods:             []computed.CachedPod{overviewPod("pod-1", "hash-current"), overviewPod("pod-2", "hash-old")},
			wantTotal:        2,
			wantNotAppliedOk: true,
			wantApplied:      1,
			wantAppliedOk:    true,
			wantOutOfDate:    1,
			wantOutOfDateOk:  false,
		},
		{
			name:             "agent enabled and a pod was never injected",
			ic:               overviewIC(true, "hash-current", false),
			pods:             []computed.CachedPod{overviewPod("pod-1", "hash-current"), overviewPod("pod-2", "")},
			wantTotal:        2,
			wantNotApplied:   1,
			wantNotAppliedOk: false,
			wantApplied:      1,
			wantAppliedOk:    true,
			wantOutOfDateOk:  true,
		},
		{
			// A "no restart required" agent is enabled in pods that have no
			// odigos label, so those pods count as applied instead of missing.
			name:             "optional pod manifest injection counts unlabelled pods as applied",
			ic:               overviewIC(true, "hash-current", true),
			pods:             []computed.CachedPod{overviewPod("pod-1", ""), overviewPod("pod-2", "")},
			wantTotal:        2,
			wantNotAppliedOk: true,
			wantApplied:      2,
			wantAppliedOk:    true,
			wantOutOfDateOk:  true,
		},
		{
			// Optional injection only applies while the agent is enabled, so a
			// disabled workload's unlabelled pods are back to "not applied".
			name:             "optional pod manifest injection with the agent disabled",
			ic:               overviewIC(false, "hash-current", true),
			pods:             []computed.CachedPod{overviewPod("pod-1", ""), overviewPod("pod-2", "")},
			wantTotal:        2,
			wantNotApplied:   2,
			wantNotAppliedOk: true,
			wantAppliedOk:    true,
			wantOutOfDateOk:  true,
		},
		{
			// With no desired hash yet, a labelled pod cannot be out of date.
			name:             "agent enabled before a hash was computed",
			ic:               overviewIC(true, "", false),
			pods:             []computed.CachedPod{overviewPod("pod-1", "hash-old")},
			wantTotal:        1,
			wantNotAppliedOk: true,
			wantApplied:      1,
			wantAppliedOk:    true,
			wantOutOfDateOk:  true,
		},
		{
			name: "mid rollout with all three buckets",
			ic:   overviewIC(true, "hash-current", false),
			pods: []computed.CachedPod{
				overviewPod("pod-1", "hash-current"),
				overviewPod("pod-2", "hash-old"),
				overviewPod("pod-3", ""),
			},
			wantTotal:        3,
			wantNotApplied:   1,
			wantNotAppliedOk: false,
			wantApplied:      1,
			wantAppliedOk:    true,
			wantOutOfDate:    1,
			wantOutOfDateOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePodsManifestInjectionOverview(tt.ic, tt.pods)
			if got == nil {
				t.Fatal("expected an overview, got nil")
			}
			if got.TotalPods != tt.wantTotal {
				t.Fatalf("expected %d total pods, got %d", tt.wantTotal, got.TotalPods)
			}
			if got.TotalAgentNotAppliedPods != tt.wantNotApplied {
				t.Fatalf("expected %d not applied pods, got %d", tt.wantNotApplied, got.TotalAgentNotAppliedPods)
			}
			if got.AgentNotAppliedOk != tt.wantNotAppliedOk {
				t.Fatalf("expected AgentNotAppliedOk %t, got %t", tt.wantNotAppliedOk, got.AgentNotAppliedOk)
			}
			if got.TotalAgentAppliedPods != tt.wantApplied {
				t.Fatalf("expected %d applied pods, got %d", tt.wantApplied, got.TotalAgentAppliedPods)
			}
			if got.AgentAppliedOk != tt.wantAppliedOk {
				t.Fatalf("expected AgentAppliedOk %t, got %t", tt.wantAppliedOk, got.AgentAppliedOk)
			}
			if got.TotalAgentOutOfDatePods != tt.wantOutOfDate {
				t.Fatalf("expected %d out of date pods, got %d", tt.wantOutOfDate, got.TotalAgentOutOfDatePods)
			}
			if got.AgentOutOfDateOk != tt.wantOutOfDateOk {
				t.Fatalf("expected AgentOutOfDateOk %t, got %t", tt.wantOutOfDateOk, got.AgentOutOfDateOk)
			}
		})
	}
}
