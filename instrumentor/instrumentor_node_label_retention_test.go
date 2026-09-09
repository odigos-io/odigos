package instrumentor

import (
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
)

// The presence of the env var is what registers the instrumented-nodes
// controllers, and its value is a helm value that reaches the instrumentor
// unvalidated.
func TestParseFirstInstrumentedPodAtNodeLabelRetention(t *testing.T) {
	tests := []struct {
		name              string
		value             string
		unset             bool
		expectedEnabled   bool
		expectedRetention time.Duration
	}{
		{
			name:  "unset keeps the node labels feature off",
			unset: true,
		},
		{
			name:  "an empty value keeps the node labels feature off",
			value: "",
		},
		{
			name:              "a duration is used as the retention",
			value:             "5m",
			expectedEnabled:   true,
			expectedRetention: 5 * time.Minute,
		},
		{
			name:              "zero retention removes the label as soon as the last pod is gone",
			value:             "0s",
			expectedEnabled:   true,
			expectedRetention: 0,
		},
		{
			name:              "a value that is not a duration falls back to the default retention",
			value:             "5 minutes",
			expectedEnabled:   true,
			expectedRetention: 5 * time.Minute,
		},
		{
			name:              "a negative retention is clamped to zero",
			value:             "-10m",
			expectedEnabled:   true,
			expectedRetention: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.unset {
				t.Setenv(k8sconsts.FirstInstrumentedPodAtNodeLabelRetentionEnvVar, tt.value)
			}

			enabled, retention := parseFirstInstrumentedPodAtNodeLabelRetention()

			assert.Equal(t, tt.expectedEnabled, enabled)
			assert.Equal(t, tt.expectedRetention, retention)
		})
	}
}
