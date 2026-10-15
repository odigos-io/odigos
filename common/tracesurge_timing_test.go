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
		{"bounds", &SamplingConfiguration{TraceSurge: &TraceSurgeConfiguration{EvaluationWindow: "5m", EvaluationInterval: "2s"}}, 5 * time.Minute, 2 * time.Second},
		{"past the window insights reads", &SamplingConfiguration{TraceSurge: &TraceSurgeConfiguration{EvaluationWindow: "10m"}}, 5 * time.Minute, 10 * time.Second},
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

func TestTraceSurgesEnabled(t *testing.T) {
	on, off := true, false
	cases := map[string]*SamplingConfiguration{
		"no sampling":       nil,
		"no trace surge":    {},
		"enabled unset":     {TraceSurge: &TraceSurgeConfiguration{}},
		"enabled false":     {TraceSurge: &TraceSurgeConfiguration{Enabled: &off}},
		"enabled true (on)": {TraceSurge: &TraceSurgeConfiguration{Enabled: &on}},
	}
	for name, s := range cases {
		if got, want := TraceSurgesEnabled(s), name == "enabled true (on)"; got != want {
			t.Errorf("%s: TraceSurgesEnabled = %v, want %v", name, got, want)
		}
	}
}
