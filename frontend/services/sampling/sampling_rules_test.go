package sampling

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestSurgeAllowed(t *testing.T) {
	on, off := true, false
	config := func(surges, insights *bool) *common.OdigosConfiguration {
		cfg := &common.OdigosConfiguration{Insights: &common.InsightsConfiguration{Enabled: insights}}
		if surges != nil {
			cfg.Sampling = &common.SamplingConfiguration{TraceSurge: &common.TraceSurgeConfiguration{Enabled: surges}}
		}
		return cfg
	}

	require.ErrorIs(t, SurgeAllowed(nil), errTraceSurgesOff)
	require.ErrorIs(t, SurgeAllowed(config(nil, &on)), errTraceSurgesOff)
	require.ErrorIs(t, SurgeAllowed(config(&off, &on)), errTraceSurgesOff)
	require.ErrorIs(t, SurgeAllowed(config(&off, &off)), errTraceSurgesOff)
	require.ErrorIs(t, SurgeAllowed(config(&on, &off)), errInsightsOff)
	require.ErrorIs(t, SurgeAllowed(config(&on, nil)), errInsightsOff)
	noInsights := config(&on, &on)
	noInsights.Insights = nil
	require.ErrorIs(t, SurgeAllowed(noInsights), errInsightsOff)
	require.NoError(t, SurgeAllowed(config(&on, &on)))

	require.Contains(t, errTraceSurgesOff.Error(), "sampling.traceSurge.enabled")
	require.Contains(t, errInsightsOff.Error(), "Odigos Insights")
}

func TestHeadSamplingRule(t *testing.T) {
	surge := &model.HeadSamplingSurgeInput{IncludeDownstream: true, Evaluation: surgeEvaluationPerService, Metric: string(v1alpha1.TraceSurgeMetricErrorRate),
		Threshold: 5, SustainedSeconds: 60, MinimumRequests: 100, BoostPercent: 80, RecoveryThreshold: 2, RecoverySeconds: 300, MinimumBoostSeconds: 300}
	input := model.NoisyOperationRuleInput{PercentageAtMost: graphql.OmittableOf(float64Ptr(1))}

	created, err := headSamplingRule(nil, input, surge, "")
	require.NoError(t, err)
	require.NotEmpty(t, created.Surge.Version)
	first := created.Surge.Version

	_, err = headSamplingRule(nil, input, surge, first)
	require.ErrorIs(t, err, errSurgeChanged, "a new rule has no surge to start from")

	_, err = headSamplingRule(&created, input, nil, "")
	require.ErrorIs(t, err, errSurgeChanged, "a removal must start from the stored surge")
	_, err = headSamplingRule(&created, input, nil, "stale")
	require.ErrorIs(t, err, errSurgeChanged)

	removed, err := headSamplingRule(&created, input, nil, first)
	require.NoError(t, err)
	require.Nil(t, removed.Surge)

	_, err = headSamplingRule(&removed, input, surge, first)
	require.ErrorIs(t, err, errSurgeChanged, "an edit of the removed surge must not apply to the rule without one")
	recreated, err := headSamplingRule(&removed, input, surge, "")
	require.NoError(t, err)
	require.NotEqual(t, first, recreated.Surge.Version, "a version is never stored again")

	edited, err := headSamplingRule(&recreated, input, surge, recreated.Surge.Version)
	require.NoError(t, err)
	require.NotEqual(t, recreated.Surge.Version, edited.Surge.Version)
	_, err = headSamplingRule(&edited, input, surge, recreated.Surge.Version)
	require.ErrorIs(t, err, errSurgeChanged)
}

func TestValidateSurgeRecoveryThreshold(t *testing.T) {
	surge := model.HeadSamplingSurgeInput{IncludeDownstream: true, Evaluation: surgeEvaluationPerService, Metric: string(v1alpha1.TraceSurgeMetricLatencyP95),
		Threshold: 500, SustainedSeconds: 60, MinimumRequests: 100, BoostPercent: 80, RecoveryThreshold: 500, RecoverySeconds: 300, MinimumBoostSeconds: 300}
	require.NoError(t, validateSurge(&surge, 1), "the evaluator recovers at or below the recovery threshold, so it may equal the threshold")

	surge.RecoveryThreshold = 500.5
	require.Error(t, validateSurge(&surge, 1))
	surge.RecoveryThreshold = -1
	require.Error(t, validateSurge(&surge, 1))
}
