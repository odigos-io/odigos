package config_test

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configurableProcessor is a ProcessorConfigurer mock that lets a test set the type and signals.
type configurableProcessor struct {
	id      string
	pType   string
	signals []common.ObservabilitySignal
}

func (p configurableProcessor) GetID() string   { return p.id }
func (p configurableProcessor) GetType() string { return p.pType }
func (p configurableProcessor) GetConfig() (config.GenericMap, error) {
	return config.GenericMap{}, nil
}
func (p configurableProcessor) GetSignals() []common.ObservabilitySignal { return p.signals }
func (p configurableProcessor) GetOrderHint() int                        { return 0 }

// profilesDestination is a Pyroscope-typed destination so the real Pyroscope configer registers a
// "profiles/pyroscope-<id>" pipeline via addProfilesPipeline.
type profilesDestination struct{ id string }

func (d profilesDestination) GetID() string                   { return d.id }
func (d profilesDestination) GetType() common.DestinationType { return common.PyroscopeDestinationType }
func (d profilesDestination) GetConfig() map[string]string {
	return map[string]string{"PYROSCOPE_URL": "pyroscope.example.com:4040"}
}
func (d profilesDestination) GetSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.ProfilesObservabilitySignal}
}

// TestGatewayProfilesProcessorsWired renders a full gateway config and asserts profiles flow through
// the same router path as traces/metrics/logs: a "profiles/in" root pipeline runs the processors once
// and exports to odigosrouterconnector/profiles, and the destination pipeline receives from its
// forward connector (no batch — unsupported for profiles).
func TestGatewayProfilesProcessorsWired(t *testing.T) {
	profilesSig := []common.ObservabilitySignal{common.ProfilesObservabilitySignal}

	processors := []config.ProcessorConfigurer{
		configurableProcessor{id: "addcluster", pType: "resource", signals: profilesSig},
		configurableProcessor{id: "rename", pType: "transform", signals: profilesSig},
		configurableProcessor{ // traces-only: must not leak into profiles
			id: "traces-only", pType: "resource",
			signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}

	gatewayOptions := pipelinegen.GatewayConfigOptions{OdigosNamespace: "odigos-system"}

	rendered, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{profilesDestination{id: "p1"}},
		processors,
		func(c *config.Config, _ []string, _ []string) error { return nil },
		nil, &gatewayOptions,
	)
	require.NoError(t, err)
	require.NoError(t, statuses.Destination["p1"])
	require.Contains(t, signals, common.ProfilesObservabilitySignal)

	// Root pipeline: otlp -> [resource/odigos-version, processors] -> odigosrouterconnector/profiles.
	root, ok := rendered.Service.Pipelines["profiles/in"]
	require.True(t, ok, "expected profiles/in root pipeline")
	assert.Equal(t, []string{"otlp"}, root.Receivers)
	assert.Contains(t, root.Processors, "resource/addcluster")
	assert.Contains(t, root.Processors, "transform/rename")
	assert.NotContains(t, root.Processors, "resource/traces-only")
	assert.Equal(t, []string{"odigosrouterconnector/profiles"}, root.Exporters)

	// Destination pipeline: receives from its forward connector, exports, no batch processor.
	dest, ok := rendered.Service.Pipelines["profiles/pyroscope-p1"]
	require.True(t, ok)
	assert.Contains(t, dest.Receivers, "forward/profiles/pyroscope-p1")
	assert.NotContains(t, dest.Processors, "batch")
}

// TestGatewayNoProfilesProcessorsWhenNoneSelected: with no PROFILES action, the profiles root pipeline
// carries only resource/odigos-version (no user processors).
func TestGatewayNoProfilesProcessorsWhenNoneSelected(t *testing.T) {
	processors := []config.ProcessorConfigurer{
		configurableProcessor{
			id: "rename", pType: "transform",
			signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}
	gatewayOptions := pipelinegen.GatewayConfigOptions{OdigosNamespace: "odigos-system"}

	rendered, err, _, _ := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{profilesDestination{id: "p1"}},
		processors,
		nil, nil, &gatewayOptions,
	)
	require.NoError(t, err)

	root, ok := rendered.Service.Pipelines["profiles/in"]
	require.True(t, ok)
	assert.Equal(t, []string{"resource/odigos-version"}, root.Processors)
	assert.NotContains(t, root.Processors, "transform/rename")
}
