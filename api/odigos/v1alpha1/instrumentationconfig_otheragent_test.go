package v1alpha1

import "testing"

func TestDetectedOtherAgents(t *testing.T) {
	t.Run("prefers plural OtherAgents", func(t *testing.T) {
		rd := &RuntimeDetailsByContainer{
			OtherAgents: []OtherAgent{{Name: "Datadog Agent"}, {Name: "New Relic Agent"}},
			OtherAgent:  &OtherAgent{Name: "ignored legacy"},
		}
		got := rd.DetectedOtherAgents()
		if len(got) != 2 || got[0].Name != "Datadog Agent" || got[1].Name != "New Relic Agent" {
			t.Fatalf("DetectedOtherAgents() = %v, want plural list", got)
		}
	})

	t.Run("falls back to deprecated singular OtherAgent", func(t *testing.T) {
		rd := &RuntimeDetailsByContainer{
			OtherAgent: &OtherAgent{Name: "Datadog Agent"},
		}
		got := rd.DetectedOtherAgents()
		if len(got) != 1 || got[0].Name != "Datadog Agent" {
			t.Fatalf("DetectedOtherAgents() = %v, want legacy singular agent", got)
		}
	})

	t.Run("empty when neither field set", func(t *testing.T) {
		if got := (&RuntimeDetailsByContainer{}).DetectedOtherAgents(); got != nil {
			t.Fatalf("DetectedOtherAgents() = %v, want nil", got)
		}
		if got := (*RuntimeDetailsByContainer)(nil).DetectedOtherAgents(); got != nil {
			t.Fatalf("nil receiver DetectedOtherAgents() = %v, want nil", got)
		}
	})
}
