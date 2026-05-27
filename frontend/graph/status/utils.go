package status

import "github.com/odigos-io/odigos/frontend/graph/model"

// givin a desired state progress enum, return a value to determine the order of severity.
// the lower the number, the more sever the state is
func desiredStateProgressSeverity(desiredStateProgress model.DesiredStateProgress) int {
	switch desiredStateProgress {
	case model.DesiredStateProgressError:
		return 0
	case model.DesiredStateProgressFailure:
		return 10
	case model.DesiredStateProgressNotice:
		return 20
	case model.DesiredStateProgressPending:
		return 30
	case model.DesiredStateProgressWaiting:
		return 40
	case model.DesiredStateProgressUnsupported:
		return 50
	case model.DesiredStateProgressDisabled:
		return 60
	case model.DesiredStateProgressSuccess:
		return 70
	case model.DesiredStateProgressIrrelevant:
		return 80
	case model.DesiredStateProgressUnknown:
		return 90
	}
	// should not happen, only as a fallback or if forgotten in the future.
	return 1000
}

func AggregateConditionsBySeverity(conditions []*model.DesiredConditionStatus) *model.DesiredConditionStatus {
	var mostSevereCondition *model.DesiredConditionStatus
	for _, condition := range conditions {
		if condition == nil {
			continue
		}
		if mostSevereCondition == nil || desiredStateProgressSeverity(condition.Status) < desiredStateProgressSeverity(mostSevereCondition.Status) {
			mostSevereCondition = condition
		}
	}
	return mostSevereCondition
}
