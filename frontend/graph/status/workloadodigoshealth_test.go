package status

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

// The gate is applied before any condition is calculated, so getting it wrong
// either hides a real health problem on every static pod or makes static pods
// look permanently unsupported on the tier that does support them.
func TestStaticPodEnterpriseFeatureHealthStatusFiresOnlyForStaticPodsOffOnPrem(t *testing.T) {
	for _, kind := range model.AllK8sResourceKind {
		for _, tier := range model.AllTier {
			wantOverride := kind == model.K8sResourceKindStaticPod && tier != model.TierOnprem

			got := StaticPodEnterpriseFeatureHealthStatus(kind, tier)

			if !wantOverride {
				if got != nil {
					t.Fatalf("kind %q tier %q: expected no override, got %+v", kind, tier, got)
				}
				continue
			}

			if got == nil {
				t.Fatalf("kind %q tier %q: expected an override, got nil", kind, tier)
			}
			if got.Name != "WorkloadOdigosHealth" {
				t.Fatalf("kind %q tier %q: expected name %q, got %q", kind, tier, "WorkloadOdigosHealth", got.Name)
			}
			if got.Status != model.DesiredStateProgressUnsupported {
				t.Fatalf("kind %q tier %q: expected status %q, got %q", kind, tier, model.DesiredStateProgressUnsupported, got.Status)
			}
			if reasonOf(t, got) != "EnterpriseFeature" {
				t.Fatalf("kind %q tier %q: expected reason %q, got %q", kind, tier, "EnterpriseFeature", reasonOf(t, got))
			}
			if got.Message != "Static pod instrumentation is an enterprise (on-prem) feature" {
				t.Fatalf("kind %q tier %q: unexpected message %q", kind, tier, got.Message)
			}
		}
	}
}

// The community and cloud tiers must both be gated; the gate is written as
// "tier != onprem" so a tier added to the schema is gated by default.
func TestStaticPodEnterpriseFeatureHealthStatusGatesEveryNonOnPremTier(t *testing.T) {
	gated := 0
	for _, tier := range model.AllTier {
		if StaticPodEnterpriseFeatureHealthStatus(model.K8sResourceKindStaticPod, tier) != nil {
			gated++
		}
	}
	if gated != len(model.AllTier)-1 {
		t.Fatalf("expected every tier but onprem (%d of %d) to be gated, got %d", len(model.AllTier)-1, len(model.AllTier), gated)
	}
}
