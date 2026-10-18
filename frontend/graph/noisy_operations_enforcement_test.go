package graph

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func neEnforcementPtr(v model.NoisyOperationsEnforcement) *model.NoisyOperationsEnforcement {
	return &v
}

func neOdigosEnforcementPtr(v commonapisampling.NoisyOperationsEnforcement) *commonapisampling.NoisyOperationsEnforcement {
	return &v
}

// The GraphQL enum and the odigos config value are two independent string enums that the
// converters bridge by hand, and frontend/services bridges with a plain cast. Both only work
// while the two spellings are identical, so pin the pairing rather than deriving one from
// the other.
func TestNoisyOperationsEnforcementModelValuesMatchTheOdigosConfigValues(t *testing.T) {
	t.Parallel()

	require.Equal(t, "auto", string(model.NoisyOperationsEnforcementAuto))
	require.Equal(t, "always", string(model.NoisyOperationsEnforcementAlways))
	require.Equal(t, string(commonapisampling.NoisyOperationsEnforcementAuto), string(model.NoisyOperationsEnforcementAuto))
	require.Equal(t, string(commonapisampling.NoisyOperationsEnforcementAlways), string(model.NoisyOperationsEnforcementAlways))
}

// Both converters route through a switch with a default arm, so a member added to the schema
// and forgotten in the switch would silently be stored (and served) as "auto".
func TestNoisyOperationsEnforcementConvertersRoundTripEveryEnumMember(t *testing.T) {
	t.Parallel()

	stored := make(map[commonapisampling.NoisyOperationsEnforcement]model.NoisyOperationsEnforcement, len(model.AllNoisyOperationsEnforcement))
	for _, member := range model.AllNoisyOperationsEnforcement {
		fromModel := noisyOperationsEnforcementFromModel(neEnforcementPtr(member))
		require.NotNil(t, fromModel)
		require.Equal(t, member, noisyOperationsEnforcementToModel(*fromModel),
			"%q does not survive the odigos config round trip", member)
		stored[*fromModel] = member
	}

	// a member swallowed by a default arm collapses onto another member's stored value
	require.Len(t, stored, len(model.AllNoisyOperationsEnforcement))
}

// An absent dropdown value must stay absent: writing a value would pin the UI overlay and
// shadow whatever the helm chart configured.
func TestNoisyOperationsEnforcementFromModelKeepsAnUnsetValueUnset(t *testing.T) {
	t.Parallel()

	require.Nil(t, noisyOperationsEnforcementFromModel(nil))

	config := &model.SamplingConfigInput{TailSampling: &model.TailSamplingConfigInput{}}
	converted := convertSamplingConfigInputToOdigosConfig(config)
	require.NotNil(t, converted.TailSampling)
	require.Nil(t, converted.TailSampling.NoisyOperationsEnforcement)
}

// toModel falls back to "auto" for anything it does not recognise. gqlgen marshals enums
// without validating them, so the fallback has to be a member the schema actually declares.
func TestNoisyOperationsEnforcementToModelFallsBackToAValidAutoValue(t *testing.T) {
	t.Parallel()

	for _, stored := range []commonapisampling.NoisyOperationsEnforcement{"", "Always", "ALWAYS", "never"} {
		served := noisyOperationsEnforcementToModel(stored)
		require.Equal(t, model.NoisyOperationsEnforcementAuto, served, "stored value %q", stored)
		require.True(t, served.IsValid())
	}

	require.Equal(t, model.NoisyOperationsEnforcementAlways,
		noisyOperationsEnforcementToModel(commonapisampling.NoisyOperationsEnforcementAlways))
}

func TestConvertOdigosConfigToSamplingConfigSurfacesNoisyOperationsEnforcement(t *testing.T) {
	t.Parallel()

	waitDuration := "45s"
	disabled := false
	tailSampling := func() *commonapisampling.TailSamplingConfiguration {
		return &commonapisampling.TailSamplingConfiguration{
			Disabled:                     &disabled,
			TraceAggregationWaitDuration: &waitDuration,
		}
	}

	withoutEnforcement := convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{TailSampling: tailSampling()},
	})
	require.NotNil(t, withoutEnforcement.TailSampling)
	require.Nil(t, withoutEnforcement.TailSampling.NoisyOperationsEnforcement)

	configured := tailSampling()
	configured.NoisyOperationsEnforcement = neOdigosEnforcementPtr(commonapisampling.NoisyOperationsEnforcementAlways)

	withEnforcement := convertOdigosConfigToSamplingConfig(&common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{TailSampling: configured},
	})

	want := *withoutEnforcement.TailSampling
	want.NoisyOperationsEnforcement = neEnforcementPtr(model.NoisyOperationsEnforcementAlways)
	require.Equal(t, &want, withEnforcement.TailSampling)
}

func TestConvertSamplingConfigInputToOdigosConfigSurfacesNoisyOperationsEnforcement(t *testing.T) {
	t.Parallel()

	waitDuration := "45s"
	disabled := false
	tailSamplingInput := func() *model.TailSamplingConfigInput {
		return &model.TailSamplingConfigInput{
			Disabled:                     &disabled,
			TraceAggregationWaitDuration: &waitDuration,
		}
	}

	withoutEnforcement := convertSamplingConfigInputToOdigosConfig(&model.SamplingConfigInput{
		TailSampling: tailSamplingInput(),
	})

	configured := tailSamplingInput()
	configured.NoisyOperationsEnforcement = neEnforcementPtr(model.NoisyOperationsEnforcementAlways)

	withEnforcement := convertSamplingConfigInputToOdigosConfig(&model.SamplingConfigInput{
		TailSampling: configured,
	})

	want := *withoutEnforcement.TailSampling
	want.NoisyOperationsEnforcement = neOdigosEnforcementPtr(commonapisampling.NoisyOperationsEnforcementAlways)
	require.Equal(t, &want, withEnforcement.TailSampling)
}
