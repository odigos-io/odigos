package rtml

import "testing"

func TestMemoryLimitMetricsAreReadable(t *testing.T) {
	stats := GetMemLimitRelatedStats()
	if stats.MemoryLimit == 0 {
		t.Fatal("expected Go memory limit metric to be available")
	}
	if stats.HeapGoal == 0 {
		t.Fatal("expected Go heap goal metric to be available")
	}

	_ = IsMemLimitReached()
}
