package status

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	actionstatus "github.com/odigos-io/odigos/status/action/generated"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertActionConditionsToStatuses_MatchingGeneration(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:               actionstatus.TransformedToProcessorType,
			Status:             metav1.ConditionTrue,
			Reason:             actionstatus.TransformedToProcessorProcessorCreated.Name,
			Message:            "ok",
			ObservedGeneration: 3,
		},
	}

	got := ConvertActionConditionsToStatuses(conditions, 3)
	if len(got) != 1 {
		t.Fatalf("expected 1 status, got %d", len(got))
	}
	if got[0].Status != model.DesiredStateProgressSuccess {
		t.Fatalf("expected Success, got %q", got[0].Status)
	}
	if got[0].Name != actionstatus.TransformedToProcessorType {
		t.Fatalf("expected name %q, got %q", actionstatus.TransformedToProcessorType, got[0].Name)
	}
}

func TestConvertActionConditionsToStatuses_StaleGeneration(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:               actionstatus.AddedToCollectorConfigType,
			Status:             metav1.ConditionTrue,
			Reason:             "ConfigUpdated_ClusterGateway",
			Message:            "stale success",
			ObservedGeneration: 2,
		},
	}

	got := ConvertActionConditionsToStatuses(conditions, 3)
	if len(got) != 1 {
		t.Fatalf("expected 1 status, got %d", len(got))
	}
	if got[0].Status != model.DesiredStateProgressWaiting {
		t.Fatalf("expected Waiting, got %q", got[0].Status)
	}
	if reasonOf(t, got[0]) != actionstatus.AddedToCollectorConfigWaitingForReconcile.Title {
		t.Fatalf("expected reason %q, got %q", actionstatus.AddedToCollectorConfigWaitingForReconcile.Title, reasonOf(t, got[0]))
	}
	if got[0].Message != actionstatus.AddedToCollectorConfigWaitingForReconcile.Message {
		t.Fatalf("expected message %q, got %q", actionstatus.AddedToCollectorConfigWaitingForReconcile.Message, got[0].Message)
	}
	if got[0].Name != actionstatus.AddedToCollectorConfigType {
		t.Fatalf("expected name %q, got %q", actionstatus.AddedToCollectorConfigType, got[0].Name)
	}
}
