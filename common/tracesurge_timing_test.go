package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTraceSurgeTiming(t *testing.T) {
	tests := []struct {
		name             string
		sampling         *SamplingConfiguration
		window, interval time.Duration
	}{
		{"unset", nil, time.Minute, 10 * time.Second},
		{"no trace surge settings", &SamplingConfiguration{}, time.Minute, 10 * time.Second},
		{"set", &SamplingConfiguration{TraceSurge: &TraceSurgeConfiguration{EvaluationWindow: "20s", EvaluationInterval: "5s"}}, 20 * time.Second, 5 * time.Second},
		{"bounds", &SamplingConfiguration{TraceSurge: &TraceSurgeConfiguration{EvaluationWindow: "1h", EvaluationInterval: "2s"}}, time.Hour, 2 * time.Second},
		{"out of range", &SamplingConfiguration{TraceSurge: &TraceSurgeConfiguration{EvaluationWindow: "5s", EvaluationInterval: "2m"}}, time.Minute, 10 * time.Second},
		{"invalid", &SamplingConfiguration{TraceSurge: &TraceSurgeConfiguration{EvaluationWindow: "soon", EvaluationInterval: "-1s"}}, time.Minute, 10 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window, interval := TraceSurgeTiming(tt.sampling)
			assert.Equal(t, tt.window, window)
			assert.Equal(t, tt.interval, interval)
		})
	}
}
