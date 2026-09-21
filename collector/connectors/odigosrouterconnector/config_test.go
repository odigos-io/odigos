package odigosrouterconnector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	collectorpipeline "go.opentelemetry.io/collector/pipeline"
	xpipeline "go.opentelemetry.io/collector/pipeline/xpipeline"
)

func TestConfigValidate(t *testing.T) {
	// the extension is what resolves a workload to its data streams; without it the connector has
	// no routing decision to make at all.
	require.ErrorContains(t, (&Config{}).Validate(), "odigos_config_extension is required")
	require.ErrorContains(t, createDefaultConfig().(*Config).Validate(), "odigos_config_extension is required")

	require.NoError(t, routerConfig().Validate())
}

func TestFactory_SupportsAllFourSignals(t *testing.T) {
	factory := NewFactory()

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesStability())
	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsToMetricsStability())
	assert.Equal(t, component.StabilityLevelAlpha, factory.LogsToLogsStability())
	assert.Equal(t, component.StabilityLevelAlpha, factory.ProfilesToProfilesStability())
}

// The generated lifecycle test only exercises traces, metrics and logs through the factory, so this
// covers the profiles wiring end to end.
func TestFactory_CreateProfilesToProfiles(t *testing.T) {
	routed := new(consumertest.ProfilesSink)
	router := xconnector.NewProfilesRouter(map[collectorpipeline.ID]xconsumer.Profiles{
		collectorpipeline.NewIDWithName(xpipeline.SignalProfiles, routerStreamA): routed,
	})

	c, err := NewFactory().CreateProfilesToProfiles(t.Context(), connectortest.NewNopSettings(typ), routerConfig(), router)
	require.NoError(t, err)

	startRouter(t, c, newRouterExtension(map[string][]string{"checkout": {routerStreamA}}))
	require.NoError(t, c.ConsumeProfiles(t.Context(), newRouterProfiles("checkout")))

	assert.Equal(t, [][]string{{"checkout"}}, profilesBatches(routed))
}
