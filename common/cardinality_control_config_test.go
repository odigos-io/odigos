package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Live traffic learning is an opt-in enterprise feature whose "on" switch has to
// travel four pointer hops from the OdigosConfiguration root. Every consumer
// (the gateway exporter, and the instrumentor learning cycle behind it) decides
// whether to install an enterprise-only component from this one answer, so each
// hop gets its own fixture: a fixture that already fails an earlier nil check
// cannot see a later one being dropped.
func TestUrlTemplatizationLiveTrafficLearningActive(t *testing.T) {
	assert.False(t, UrlTemplatizationLiveTrafficLearningActive(nil))
	assert.False(t, UrlTemplatizationLiveTrafficLearningActive(&CardinalityControlConfiguration{}),
		"cardinalityControl present but urlTemplatization unset must stay off")

	c := &CardinalityControlConfiguration{UrlTemplatization: &UrlTemplatizationCardinalityControlConfiguration{}}
	assert.False(t, UrlTemplatizationLiveTrafficLearningActive(c),
		"urlTemplatization present but liveTrafficLearning unset must stay off")

	c.UrlTemplatization.LiveTrafficLearning = &LiveTrafficLearningConfiguration{}
	assert.False(t, UrlTemplatizationLiveTrafficLearningActive(c),
		"an unset enabled flag must stay off - the feature deploys odigos-cache, so it cannot default on")

	// A block that carries tunables but no opt-in flag is the shape a user lands
	// on after editing the learning settings in the UI without a helm change:
	// configuring the feature must not be what switches it on.
	c.UrlTemplatization.LiveTrafficLearning.LearningInterval = "15s"
	maxPaths := 100
	c.UrlTemplatization.LiveTrafficLearning.MaxExamplePathsPerWorkload = &maxPaths
	assert.False(t, UrlTemplatizationLiveTrafficLearningActive(c),
		"tunables without an explicit enabled flag must stay off")

	off := false
	c.UrlTemplatization.LiveTrafficLearning.Enabled = &off
	assert.False(t, UrlTemplatizationLiveTrafficLearningActive(c))

	on := true
	c.UrlTemplatization.LiveTrafficLearning.Enabled = &on
	assert.True(t, UrlTemplatizationLiveTrafficLearningActive(c))
}

func TestOdigosConfiguration_UrlTemplatizationLiveTrafficLearningEnabled(t *testing.T) {
	assert.False(t, (*OdigosConfiguration)(nil).UrlTemplatizationLiveTrafficLearningEnabled())

	cfg := &OdigosConfiguration{}
	assert.False(t, cfg.UrlTemplatizationLiveTrafficLearningEnabled(),
		"a configuration with no cardinalityControl block at all must stay off")

	on := true
	cfg.CardinalityControl = &CardinalityControlConfiguration{
		UrlTemplatization: &UrlTemplatizationCardinalityControlConfiguration{
			LiveTrafficLearning: &LiveTrafficLearningConfiguration{Enabled: &on},
		},
	}
	assert.True(t, cfg.UrlTemplatizationLiveTrafficLearningEnabled())

	// The method must keep reading its own block. Insights and profiling have the
	// same shape one field over, and all three are consulted from the same place
	// in the gateway sync.
	cfg.Insights = &InsightsConfiguration{}
	cfg.Profiling = &ProfilingConfiguration{}
	assert.True(t, cfg.UrlTemplatizationLiveTrafficLearningEnabled())
	assert.False(t, cfg.InsightsEnabled(), "an empty insights block must not inherit the learning flag")
	assert.False(t, cfg.ProfilingEnabled(), "an empty profiling block must not inherit the learning flag")
}
