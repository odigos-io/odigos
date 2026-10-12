package services

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/config"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

// the settings screen and the provenance map address this field by this exact path
const neEnforcementHelmValuePath = "sampling.tailSampling.noisyOperationsEnforcement"

func neEnforcementInput(v model.NoisyOperationsEnforcement) *model.LocalUIConfigSamplingInput {
	return &model.LocalUIConfigSamplingInput{
		TailSampling: &model.TailSamplingConfigInput{NoisyOperationsEnforcement: &v},
	}
}

// applySamplingInput casts the GraphQL enum straight to the odigos config type instead of
// mapping it, so every member the schema offers has to land on a value the instrumentor
// recognises. A member that does not is persisted and then silently ignored.
func TestApplySamplingInputWritesAnEnforcementTheInstrumentorRecognises(t *testing.T) {
	t.Parallel()

	want := map[model.NoisyOperationsEnforcement]sampling.NoisyOperationsEnforcement{
		model.NoisyOperationsEnforcementAuto:   sampling.NoisyOperationsEnforcementAuto,
		model.NoisyOperationsEnforcementAlways: sampling.NoisyOperationsEnforcementAlways,
	}
	require.Len(t, want, len(model.AllNoisyOperationsEnforcement),
		"a new NoisyOperationsEnforcement member has no expected odigos config value")

	for _, member := range model.AllNoisyOperationsEnforcement {
		t.Run(string(member), func(t *testing.T) {
			expected, declared := want[member]
			require.True(t, declared, "%q is not mapped to an odigos config value", member)

			cfg := &common.SamplingConfiguration{}
			applySamplingInput(cfg, neEnforcementInput(member))

			require.NotNil(t, cfg.TailSampling)
			require.NotNil(t, cfg.TailSampling.NoisyOperationsEnforcement)
			require.Equal(t, expected, *cfg.TailSampling.NoisyOperationsEnforcement)
		})
	}
}

// The local UI overlay is a partial update: a sampling input that does not carry the
// dropdown must leave an already stored enforcement alone, and setting the dropdown must
// not disturb the sibling tail sampling fields.
func TestApplySamplingInputUpdatesOnlyTheFieldsTheInputCarries(t *testing.T) {
	t.Parallel()

	waitDuration := "45s"
	disabled := false
	stored := func() *common.SamplingConfiguration {
		auto := sampling.NoisyOperationsEnforcementAuto
		return &common.SamplingConfiguration{
			TailSampling: &sampling.TailSamplingConfiguration{
				Disabled:                     &disabled,
				TraceAggregationWaitDuration: &waitDuration,
				NoisyOperationsEnforcement:   &auto,
			},
		}
	}

	untouched := stored()
	applySamplingInput(untouched, &model.LocalUIConfigSamplingInput{TailSampling: &model.TailSamplingConfigInput{}})
	require.Equal(t, stored(), untouched)

	updated := stored()
	applySamplingInput(updated, neEnforcementInput(model.NoisyOperationsEnforcementAlways))

	want := stored()
	always := sampling.NoisyOperationsEnforcementAlways
	want.TailSampling.NoisyOperationsEnforcement = &always
	require.NotEqual(t, stored(), want)
	require.Equal(t, want, updated)
}

func neSettingsField(t *testing.T, helmValuePath string) config.ConfigurationField {
	t.Helper()

	require.NoError(t, config.Load())

	var found []config.ConfigurationField
	for _, cfg := range config.Get() {
		for _, field := range cfg.Spec.Fields {
			if field.HelmValuePath == helmValuePath {
				found = append(found, field)
			}
		}
	}
	require.Len(t, found, 1, "expected exactly one settings field for %q", helmValuePath)
	return found[0]
}

// The settings screen renders the dropdown from the embedded descriptor, and gqlgen rejects
// any value outside the schema enum on the way back in. An option the enum does not declare
// is a dropdown entry the user can pick but never save, and a member the descriptor omits is
// one the user can never pick at all.
func TestSettingsDropdownOffersExactlyTheNoisyOperationsEnforcementEnumMembers(t *testing.T) {
	field := neSettingsField(t, neEnforcementHelmValuePath)
	require.Equal(t, "dropdown", field.ComponentType)

	options, ok := field.ComponentProps["options"].([]any)
	require.True(t, ok, "dropdown %q has no options list", field.DisplayName)

	offered := make([]model.NoisyOperationsEnforcement, 0, len(options))
	for _, option := range options {
		value, ok := option.(string)
		require.True(t, ok, "option %#v is not a string", option)

		member := model.NoisyOperationsEnforcement(value)
		require.True(t, member.IsValid(), "the settings dropdown offers %q, which the GraphQL enum rejects", value)
		offered = append(offered, member)
	}

	require.ElementsMatch(t, model.AllNoisyOperationsEnforcement, offered)
}

// The UI matches a provenance entry to a settings field by the helm value path, so the key
// recorded for an overlay has to be the path the descriptor advertises.
func TestOverlayProvenanceKeyMatchesTheNoisyOperationsEnforcementSettingsField(t *testing.T) {
	always := sampling.NoisyOperationsEnforcementAlways
	overlay := &common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			TailSampling: &sampling.TailSamplingConfiguration{NoisyOperationsEnforcement: &always},
		},
	}

	provenance := map[string]string{}
	recordOverlayProvenance(overlay, provenance, "local-ui-config")

	field := neSettingsField(t, neEnforcementHelmValuePath)
	require.Equal(t, map[string]string{field.HelmValuePath: "local-ui-config"}, provenance)
}
