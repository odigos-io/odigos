package status

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
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
					Reason:  string(podsManifestInjectionStatus.PodsManifestInjectionReasonWaitingInRolloutQueue),
					Message: "waiting for rollout",
				},
			},
		},
	}

	got := CalculatePodsManifestInjectionStatus(ic)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if len(got.ActionItems) != 1 {
		t.Fatalf("expected one action item, got %d", len(got.ActionItems))
	}
	if got.ActionItems[0].Type != model.DesiredConditionActionItemTypeRolloutWorkload {
		t.Fatalf("expected action type %q, got %q", model.DesiredConditionActionItemTypeRolloutWorkload, got.ActionItems[0].Type)
	}
	if got.ActionItems[0].UserFacingText == "" {
		t.Fatal("expected action item user-facing text")
	}
}
