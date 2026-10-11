package status

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/status"
	statuscatalog "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
)

// The order the UI expects, from most severe to least severe. Hardcoded rather
// than derived from desiredStateProgressSeverity so that a change to the
// production severity table has to be made here too.
var severityOrderMostSevereFirst = []model.DesiredStateProgress{
	model.DesiredStateProgressError,
	model.DesiredStateProgressFailure,
	model.DesiredStateProgressNotice,
	model.DesiredStateProgressPending,
	model.DesiredStateProgressWaiting,
	model.DesiredStateProgressUnsupported,
	model.DesiredStateProgressDisabled,
	model.DesiredStateProgressSuccess,
	model.DesiredStateProgressIrrelevant,
	model.DesiredStateProgressUnknown,
}

func progressCondition(name string, progress model.DesiredStateProgress) *model.DesiredConditionStatus {
	return &model.DesiredConditionStatus{
		Name:    name,
		Status:  progress,
		Message: name + " message",
	}
}

func TestDesiredStateProgressSeverityOrder(t *testing.T) {
	for i := 1; i < len(severityOrderMostSevereFirst); i++ {
		more, less := severityOrderMostSevereFirst[i-1], severityOrderMostSevereFirst[i]
		if desiredStateProgressSeverity(more) >= desiredStateProgressSeverity(less) {
			t.Fatalf("expected %q to be more severe than %q, got severities %d and %d",
				more, less, desiredStateProgressSeverity(more), desiredStateProgressSeverity(less))
		}
	}
}

// A value added to the DesiredStateProgress enum in the graphql schema but not
// to desiredStateProgressSeverity falls into the 1000 fallback, which makes it
// the least severe of all states. A new error-ish state would then be silently
// masked by every other condition in the aggregation.
func TestDesiredStateProgressSeverityCoversWholeEnum(t *testing.T) {
	ranked := make(map[model.DesiredStateProgress]struct{}, len(severityOrderMostSevereFirst))
	for _, progress := range severityOrderMostSevereFirst {
		ranked[progress] = struct{}{}
	}

	fallback := desiredStateProgressSeverity(model.DesiredStateProgress("NotAProgressValue"))

	seen := map[int]model.DesiredStateProgress{}
	for _, progress := range model.AllDesiredStateProgress {
		if _, ok := ranked[progress]; !ok {
			t.Fatalf("progress %q is missing from severityOrderMostSevereFirst; add it to the expected order and to desiredStateProgressSeverity", progress)
		}

		severity := desiredStateProgressSeverity(progress)
		if severity == fallback {
			t.Fatalf("progress %q has no explicit severity, it falls back to %d and would be masked by every other condition", progress, fallback)
		}
		if other, ok := seen[severity]; ok {
			t.Fatalf("progress %q and %q share severity %d, aggregation between them is not deterministic", progress, other, severity)
		}
		seen[severity] = progress
	}

	if len(model.AllDesiredStateProgress) != len(severityOrderMostSevereFirst) {
		t.Fatalf("expected %d ranked progress values, got %d", len(model.AllDesiredStateProgress), len(severityOrderMostSevereFirst))
	}
}

// The status catalogs are generated from yaml in the status module and their
// OdigosSeverity is cast straight to a DesiredStateProgress, with no compiler
// or codegen check that the two enums agree. A severity that is not part of the
// graphql enum reaches the UI as an invalid value and, because it is not
// ranked, is also masked by every other condition in the aggregation.
func TestDesiredStateProgressSeverityCoversTheStatusCatalogs(t *testing.T) {
	catalogs := map[string]map[string]status.Reason{
		statuscatalog.AgentEnabledType:          statuscatalog.AgentEnabledByReason,
		statuscatalog.PodsManifestInjectionType: statuscatalog.PodsManifestInjectionByReason,
	}

	fallback := desiredStateProgressSeverity(model.DesiredStateProgress("NotAProgressValue"))

	for conditionType, catalog := range catalogs {
		for reasonName, reason := range catalog {
			progress := model.DesiredStateProgress(reason.OdigosSeverity)
			if !progress.IsValid() {
				t.Fatalf("%s/%s: severity %q is not a DesiredStateProgress value", conditionType, reasonName, reason.OdigosSeverity)
			}
			if desiredStateProgressSeverity(progress) == fallback {
				t.Fatalf("%s/%s: severity %q has no severity ranking", conditionType, reasonName, reason.OdigosSeverity)
			}
		}
	}
}

func TestDesiredStateProgressSeverityUnrecognizedIsLeastSevere(t *testing.T) {
	fallback := desiredStateProgressSeverity(model.DesiredStateProgress("NotAProgressValue"))
	for _, progress := range model.AllDesiredStateProgress {
		if desiredStateProgressSeverity(progress) >= fallback {
			t.Fatalf("expected %q (severity %d) to be more severe than an unrecognized value (severity %d)",
				progress, desiredStateProgressSeverity(progress), fallback)
		}
	}
}

func TestAggregateConditionsBySeverityPicksMostSevere(t *testing.T) {
	for i := 0; i < len(severityOrderMostSevereFirst); i++ {
		for j := i + 1; j < len(severityOrderMostSevereFirst); j++ {
			more := progressCondition("more", severityOrderMostSevereFirst[i])
			less := progressCondition("less", severityOrderMostSevereFirst[j])

			// both orders, so the result cannot come from the slice position.
			if got := AggregateConditionsBySeverity([]*model.DesiredConditionStatus{less, more}); got != more {
				t.Fatalf("%q before %q: expected the %q condition, got %+v",
					severityOrderMostSevereFirst[j], severityOrderMostSevereFirst[i], severityOrderMostSevereFirst[i], got)
			}
			if got := AggregateConditionsBySeverity([]*model.DesiredConditionStatus{more, less}); got != more {
				t.Fatalf("%q before %q: expected the %q condition, got %+v",
					severityOrderMostSevereFirst[i], severityOrderMostSevereFirst[j], severityOrderMostSevereFirst[i], got)
			}
		}
	}
}

// The aggregated condition is handed straight to the UI, so it must be the
// original condition (with its reason, message and action items) and not a
// rebuilt copy carrying only the status.
func TestAggregateConditionsBySeverityReturnsTheOriginalCondition(t *testing.T) {
	failure := progressCondition("failure", model.DesiredStateProgressFailure)
	failure.ActionItems = []*model.DesiredConditionActionItem{{
		Type:       model.DesiredConditionActionItemTypeRolloutWorkload,
		ButtonText: "Rollout",
	}}

	got := AggregateConditionsBySeverity([]*model.DesiredConditionStatus{
		progressCondition("success", model.DesiredStateProgressSuccess),
		failure,
	})

	if got != failure {
		t.Fatalf("expected the original failure condition pointer, got %+v", got)
	}
}

func TestAggregateConditionsBySeveritySkipsNilConditions(t *testing.T) {
	// Callers append conditions unconditionally and several Calculate* functions
	// return nil (e.g. rollout status for an un-reconciled workload), so nil
	// entries are the normal case rather than an error.
	waiting := progressCondition("waiting", model.DesiredStateProgressWaiting)

	got := AggregateConditionsBySeverity([]*model.DesiredConditionStatus{
		nil,
		progressCondition("success", model.DesiredStateProgressSuccess),
		nil,
		waiting,
		nil,
	})

	if got != waiting {
		t.Fatalf("expected the waiting condition, got %+v", got)
	}
}

func TestAggregateConditionsBySeverityNoUsableConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions []*model.DesiredConditionStatus
	}{
		{name: "nil slice", conditions: nil},
		{name: "empty slice", conditions: []*model.DesiredConditionStatus{}},
		{name: "only nil conditions", conditions: []*model.DesiredConditionStatus{nil, nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AggregateConditionsBySeverity(tt.conditions); got != nil {
				t.Fatalf("expected nil, got %+v", got)
			}
		})
	}
}

// Equal severity keeps the first condition, which is what makes the reported
// health stable across requests instead of flapping between two conditions.
func TestAggregateConditionsBySeverityKeepsFirstOfEqualSeverity(t *testing.T) {
	first := progressCondition("first", model.DesiredStateProgressWaiting)
	second := progressCondition("second", model.DesiredStateProgressWaiting)

	if got := AggregateConditionsBySeverity([]*model.DesiredConditionStatus{first, second}); got != first {
		t.Fatalf("expected the first waiting condition, got %+v", got)
	}
	if got := AggregateConditionsBySeverity([]*model.DesiredConditionStatus{second, first}); got != second {
		t.Fatalf("expected the first waiting condition, got %+v", got)
	}
}

// Overall workload Odigos health is this aggregation over the workload's
// conditions, so every condition fed into it can single-handedly downgrade the
// workload's reported health. That is why aggregating the same signal twice, or
// aggregating a condition the workload health is not meant to include, changes
// what the UI shows even when nothing about the workload changed.
func TestAggregateConditionsBySeverityASingleNoticeMasksAllSuccesses(t *testing.T) {
	healthy := []*model.DesiredConditionStatus{
		progressCondition("runtimeDetection", model.DesiredStateProgressSuccess),
		progressCondition("agentInjectionEnabled", model.DesiredStateProgressSuccess),
		progressCondition("podsManifestInjection", model.DesiredStateProgressSuccess),
		progressCondition("processesHealth", model.DesiredStateProgressSuccess),
		progressCondition("expectingTelemetry", model.DesiredStateProgressSuccess),
	}

	got := AggregateConditionsBySeverity(healthy)
	if got == nil || got.Status != model.DesiredStateProgressSuccess {
		t.Fatalf("expected an all-success aggregation to be Success, got %+v", got)
	}

	notice := progressCondition("autoRollback", model.DesiredStateProgressNotice)
	got = AggregateConditionsBySeverity(append(healthy, notice))
	if got != notice {
		t.Fatalf("expected the single notice condition to win over every success condition, got %+v", got)
	}
}
