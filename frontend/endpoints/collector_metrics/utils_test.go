package collectormetrics

import (
	"testing"
	"time"
)

func TestCalculateThroughput(t *testing.T) {
	cases := []struct {
		name string
		diff float64
		currentTime time.Time
		prevTime time.Time
		want int64
	}{
		{name: "no diff and no time change", diff: 0, currentTime: time.Now(), prevTime: time.Now(), want: 0},
		{name: "diff of 1 in 1 second", diff: 1, currentTime: time.Now(), prevTime: time.Now().Add(-1 * time.Second), want: 1},
		{name: "diff of 1 in 2 seconds", diff: 1, currentTime: time.Now(), prevTime: time.Now().Add(-2 * time.Second), want: 0},
		{name: "diff of 0 in 1 second", diff: 0, currentTime: time.Now(), prevTime: time.Now().Add(-1 * time.Second), want: 0},
		{name: "diff of 100 in 10 seconds", diff: 100, currentTime: time.Now(), prevTime: time.Now().Add(-10 * time.Second), want: 10},
		{name: "diff of 100 in 0.1 seconds", diff: 100, currentTime: time.Now(), prevTime: time.Now().Add(-100 * time.Millisecond), want: 1000},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := calculateThroughput(c.diff, c.currentTime, c.prevTime)
			if got != c.want {
				t.Errorf("calculateThroughput(%f, %v, %v) = %d, want %d", c.diff, c.currentTime, c.prevTime, got, c.want)
			}
		})
	}

}