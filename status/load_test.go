package status

import (
	"testing"
)

func TestLoad(t *testing.T) {
	err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	statuses := Get()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 status manifests, got %d", len(statuses))
	}

	podsInjection, ok := GetStatusByType("PodsInjection")
	if !ok {
		t.Fatal("expected PodsInjection status type")
	}
	if len(podsInjection.Spec.Reasons) == 0 {
		t.Fatal("expected PodsInjection to have reasons")
	}

	rollback, ok := GetStatusByType("Rollback")
	if !ok {
		t.Fatal("expected Rollback status type")
	}
	if len(rollback.Spec.Reasons) != 7 {
		t.Fatalf("expected 7 rollback reasons, got %d", len(rollback.Spec.Reasons))
	}
}
