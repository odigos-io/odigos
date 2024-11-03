package reconcile

import (
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// TimeTracker is a utility to track the time since a given key was started
// and to determine if the time since the start is less than the maximum reconcile time
// If the time since the start is greater than the maximum reconcile time, the key is cleared
// this can be helpful to prevent a single key from being stuck in a reconcile loop

const MaxReconcileTime = 3 * time.Minute

type TimeTracker struct {
	startTimes map[types.NamespacedName]time.Time
}

func NewTimeTracker() *TimeTracker {
	return &TimeTracker{
		startTimes: make(map[types.NamespacedName]time.Time),
	}
}

func (tt *TimeTracker) Start(key types.NamespacedName) {
	if _, exists := tt.startTimes[key]; !exists {
		tt.startTimes[key] = time.Now()
	}
}

func (tt *TimeTracker) ShouldContinue(key types.NamespacedName) bool {
	return time.Since(tt.startTimes[key]) <= MaxReconcileTime
}

func (tt *TimeTracker) Clear(key types.NamespacedName) {
	delete(tt.startTimes, key)
}
