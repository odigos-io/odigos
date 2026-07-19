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
