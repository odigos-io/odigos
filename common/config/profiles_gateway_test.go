package config_test

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configurableProcessor is a ProcessorConfigurer mock that lets a test set the type and signals,
// so we can exercise the profiles-capable type gate end-to-end.
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

// TestGatewayProfilesProcessorsWired renders a full gateway config and asserts that only the
// profiles-capable Tier-1 actions (resource, transform) selecting PROFILES are prepended to every
// gateway profiles pipeline — both the destination "profiles/<dest>" pipeline and the UI "profiles"
// pipeline added by the applySelfTelemetry callback (mirroring addProfilingGatewayPipeline).
func TestGatewayProfilesProcessorsWired(t *testing.T) {
	profilesSig := []common.ObservabilitySignal{common.ProfilesObservabilitySignal}

	processors := []config.ProcessorConfigurer{
		configurableProcessor{id: "addcluster", pType: "resource", signals: profilesSig}, // AddClusterInfo
		configurableProcessor{id: "rename", pType: "transform", signals: profilesSig},    // RenameAttribute
		configurableProcessor{id: "k8s", pType: "k8sattributes", signals: profilesSig},   // excluded (built-in)
		configurableProcessor{id: "redact", pType: "redaction", signals: profilesSig},    // excluded (not capable)
		configurableProcessor{ // traces-only resource: must not leak into profiles
			id: "traces-only", pType: "resource",
			signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		},
	}

	gatewayOptions := pipelinegen.GatewayConfigOptions{OdigosNamespace: "odigos-system"}

	// Mirror addProfilingGatewayPipeline: register the UI "profiles" pipeline in the callback.
	applySelfTelemetry := func(c *config.Config, _ []string, _ []string) error {
		c.Exporters["otlp_grpc/profiles-to-ui"] = config.GenericMap{"endpoint": "ui:4317"}
		c.Service.Pipelines["profiles"] = config.Pipeline{
			Receivers: []string{"otlp"},
			Exporters: []string{"otlp_grpc/profiles-to-ui"},
		}
		return nil
	}

	rendered, err, statuses, signals := pipelinegen.CalculateGatewayConfig(
		[]config.ExporterConfigurer{profilesDestination{id: "p1"}},
		processors,
		applySelfTelemetry,
		nil, &gatewayOptions,
	)
	require.NoError(t, err)
	require.NoError(t, statuses.Destination["p1"])
	require.Contains(t, signals, common.ProfilesObservabilitySignal)

	wantProfilesProcessors := []string{"resource/addcluster", "transform/rename"}

	for _, name := range []string{"profiles", "profiles/pyroscope-p1"} {
		pl, ok := rendered.Service.Pipelines[name]
		require.True(t, ok, "expected pipeline %q to exist", name)
		assert.Equal(t, wantProfilesProcessors, pl.Processors,
			"pipeline %q should carry exactly the profiles-capable processors, in order", name)
		assert.NotContains(t, pl.Processors, "k8sattributes/k8s", "%q must not duplicate built-in k8s_attributes", name)
		assert.NotContains(t, pl.Processors, "redaction/redact", "%q must not include non-capable processors", name)
		assert.NotContains(t, pl.Processors, "resource/traces-only", "%q must not include traces-only processors", name)
	}
}

// TestGatewayNoProfilesProcessorsWhenNoneSelected ensures the profiles pipelines are left untouched
// (no processors) when no action selects the PROFILES signal — i.e. the change is inert by default.
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

	pl, ok := rendered.Service.Pipelines["profiles/pyroscope-p1"]
	require.True(t, ok)
	assert.Empty(t, pl.Processors, "profiles pipeline must stay processor-free when no PROFILES action exists")
}
