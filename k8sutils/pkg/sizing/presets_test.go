package sizing

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// every Sizing value the package declares. the length gates below force a newly declared sizing
// to be added here, which in turn makes the rest of the preset contract apply to it.
var allSizings = []Sizing{SizeSmall, SizeMedium, SizeLarge}

// presets are ordered from the smallest to the largest, which is the order the values are
// expected to grow in.
var sizingsSmallestFirst = []Sizing{SizeSmall, SizeMedium, SizeLarge}

// the numeric fields a preset is responsible for. these are also the fields the merge functions
// copy from the preset onto an existing configuration, so the lists are shared with sizing_test.go.
var (
	gatewaySizingFields = []string{
		"MinReplicas",
		"MaxReplicas",
		"RequestMemoryMiB",
		"LimitMemoryMiB",
		"RequestCPUm",
		"LimitCPUm",
		"MemoryLimiterLimitMiB",
		"MemoryLimiterSpikeLimitMiB",
		"GoMemLimitMib",
	}
	nodeSizingFields = []string{
		"RequestMemoryMiB",
		"LimitMemoryMiB",
		"RequestCPUm",
		"LimitCPUm",
		"MemoryLimiterLimitMiB",
		"MemoryLimiterSpikeLimitMiB",
		"GoMemLimitMib",
	}
)

// fields that are not part of sizing and must therefore be preserved from the user configuration.
var (
	gatewayPreservedFields = []string{
		"ServiceGraph",
		"ClusterMetricsEnabled",
		"HttpsProxyAddress",
		"NodeSelector",
		"DeploymentName",
	}
	nodePreservedFields = []string{
		"CollectorOwnMetricsPort",
		"EnableDataCompression",
		"OtlpExporterConfiguration",
		"ResourceDetectors",
	}
)

func intFieldOf(t *testing.T, cfg any, name string) int {
	t.Helper()
	field := reflect.ValueOf(cfg).FieldByName(name)
	require.True(t, field.IsValid(), "field %q does not exist on %T", name, cfg)
	return int(field.Int())
}

func fieldNamesOf(cfg any) []string {
	structType := reflect.TypeOf(cfg)
	names := make([]string, 0, structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		names = append(names, structType.Field(i).Name)
	}
	return names
}

func TestEveryDeclaredSizingHasExactlyOnePresetAndIsValid(t *testing.T) {
	require.Len(t, configs, len(allSizings), "a declared Sizing is missing a preset (or a preset has no Sizing)")
	require.Len(t, validSizings, len(allSizings), "a declared Sizing is missing from validSizings")

	for _, s := range allSizings {
		t.Run(string(s), func(t *testing.T) {
			_, hasPreset := configs[s]
			assert.True(t, hasPreset, "no preset for sizing %q", s)
			assert.True(t, IsValidSizing(string(s)), "sizing %q is not reported as valid", s)
		})
	}
}

// the values the memory limiter is configured with are what keeps a collector shedding load
// instead of being OOM killed by the kernel, so an edit to the preset table that inverts any of
// these relations is a production incident rather than a tuning change.
func TestEveryPresetKeepsTheMemoryLimiterBelowTheContainerLimit(t *testing.T) {
	for _, s := range allSizings {
		preset := configs[s]

		t.Run(string(s)+"/gateway", func(t *testing.T) {
			gw := preset.CollectorGatewayConfig
			assert.Less(t, gw.MemoryLimiterLimitMiB, gw.LimitMemoryMiB,
				"the memory limiter hard limit must be below the container memory limit")
			assert.Less(t, gw.GoMemLimitMib, gw.MemoryLimiterLimitMiB,
				"GOMEMLIMIT must be below the memory limiter hard limit")
			assert.Less(t, gw.MemoryLimiterSpikeLimitMiB, gw.MemoryLimiterLimitMiB,
				"the spike limit is a diff below the hard limit, so it must be smaller than it")
			assert.LessOrEqual(t, gw.RequestMemoryMiB, gw.LimitMemoryMiB)
			assert.LessOrEqual(t, gw.RequestCPUm, gw.LimitCPUm)
			assert.GreaterOrEqual(t, gw.MinReplicas, 1, "the gateway must always have at least one replica")
			assert.LessOrEqual(t, gw.MinReplicas, gw.MaxReplicas, "an HPA with minReplicas > maxReplicas is rejected")
		})

		t.Run(string(s)+"/node", func(t *testing.T) {
			node := preset.CollectorNodeConfig
			assert.Less(t, node.MemoryLimiterLimitMiB, node.LimitMemoryMiB,
				"the memory limiter hard limit must be below the container memory limit")
			assert.Less(t, node.GoMemLimitMib, node.MemoryLimiterLimitMiB,
				"GOMEMLIMIT must be below the memory limiter hard limit")
			assert.Less(t, node.MemoryLimiterSpikeLimitMiB, node.MemoryLimiterLimitMiB,
				"the spike limit is a diff below the hard limit, so it must be smaller than it")
			assert.LessOrEqual(t, node.RequestMemoryMiB, node.LimitMemoryMiB)
			assert.LessOrEqual(t, node.RequestCPUm, node.LimitCPUm)
		})
	}
}

// a zero value means the field is omitted from the rendered collector manifest, which silently
// drops the resource request/limit or the memory limiter setting the preset is supposed to supply.
func TestEveryPresetPopulatesEverySizingField(t *testing.T) {
	for _, s := range allSizings {
		preset := configs[s]

		for _, name := range gatewaySizingFields {
			t.Run(fmt.Sprintf("%s/gateway/%s", s, name), func(t *testing.T) {
				assert.NotZero(t, intFieldOf(t, preset.CollectorGatewayConfig, name))
			})
		}
		for _, name := range nodeSizingFields {
			t.Run(fmt.Sprintf("%s/node/%s", s, name), func(t *testing.T) {
				assert.NotZero(t, intFieldOf(t, preset.CollectorNodeConfig, name))
			})
		}
	}
}

// the two derived memory limiter values are documented on the configuration fields themselves
// ("20% of the hard limit" / "80% of the hard limit"), and the hard limit is documented as sitting
// 50MiB below the container limit. those are the numbers the collector's own docs promise, so a
// preset that stops following them is a bug even when every individual number looks plausible.
func TestEveryPresetDerivesTheMemoryLimiterFromTheContainerLimit(t *testing.T) {
	for _, s := range allSizings {
		preset := configs[s]

		t.Run(string(s)+"/gateway", func(t *testing.T) {
			gw := preset.CollectorGatewayConfig
			assert.Equal(t, gw.LimitMemoryMiB-50, gw.MemoryLimiterLimitMiB)
			assert.Equal(t, gw.MemoryLimiterLimitMiB/5, gw.MemoryLimiterSpikeLimitMiB)
			assert.Equal(t, gw.MemoryLimiterLimitMiB*4/5, gw.GoMemLimitMib)
		})

		t.Run(string(s)+"/node", func(t *testing.T) {
			node := preset.CollectorNodeConfig
			assert.Equal(t, node.LimitMemoryMiB-50, node.MemoryLimiterLimitMiB)
			assert.Equal(t, node.MemoryLimiterLimitMiB/5, node.MemoryLimiterSpikeLimitMiB)
			assert.Equal(t, node.MemoryLimiterLimitMiB*4/5, node.GoMemLimitMib)
		})
	}
}

// picking a larger preset must never give a collector less of anything.
func TestPresetsGrowWithTheSelectedSize(t *testing.T) {
	for i := 1; i < len(sizingsSmallestFirst); i++ {
		smaller := configs[sizingsSmallestFirst[i-1]]
		larger := configs[sizingsSmallestFirst[i]]
		label := fmt.Sprintf("%s<%s", sizingsSmallestFirst[i-1], sizingsSmallestFirst[i])

		for _, name := range gatewaySizingFields {
			t.Run(label+"/gateway/"+name, func(t *testing.T) {
				assert.Greater(t,
					intFieldOf(t, larger.CollectorGatewayConfig, name),
					intFieldOf(t, smaller.CollectorGatewayConfig, name))
			})
		}
		for _, name := range nodeSizingFields {
			t.Run(label+"/node/"+name, func(t *testing.T) {
				assert.Greater(t,
					intFieldOf(t, larger.CollectorNodeConfig, name),
					intFieldOf(t, smaller.CollectorNodeConfig, name))
			})
		}
	}
}

// the preset table only carries sizing, so every other field must stay at its zero value there.
// otherwise a preset would silently override a user setting such as the gateway deployment name.
func TestPresetsDoNotCarryNonSizingConfiguration(t *testing.T) {
	for _, s := range allSizings {
		preset := configs[s]

		for _, name := range gatewayPreservedFields {
			t.Run(fmt.Sprintf("%s/gateway/%s", s, name), func(t *testing.T) {
				field := reflect.ValueOf(preset.CollectorGatewayConfig).FieldByName(name)
				require.True(t, field.IsValid())
				assert.True(t, field.IsZero(), "preset %q sets the non-sizing field %q", s, name)
			})
		}
		for _, name := range nodePreservedFields {
			t.Run(fmt.Sprintf("%s/node/%s", s, name), func(t *testing.T) {
				field := reflect.ValueOf(preset.CollectorNodeConfig).FieldByName(name)
				require.True(t, field.IsValid())
				assert.True(t, field.IsZero(), "preset %q sets the non-sizing field %q", s, name)
			})
		}
	}
}

// the merge functions copy a hand maintained list of fields, so a field added to either collector
// configuration has to be classified as "comes from the preset" or "preserved from the user".
// leaving it unclassified is how a new sizing field silently stops being applied.
func TestEveryCollectorConfigurationFieldIsClassifiedAsSizingOrPreserved(t *testing.T) {
	t.Run("gateway", func(t *testing.T) {
		assert.ElementsMatch(t,
			fieldNamesOf(common.CollectorGatewayConfiguration{}),
			append(append([]string{}, gatewaySizingFields...), gatewayPreservedFields...))
	})

	t.Run("node", func(t *testing.T) {
		assert.ElementsMatch(t,
			fieldNamesOf(common.CollectorNodeConfiguration{}),
			append(append([]string{}, nodeSizingFields...), nodePreservedFields...))
	})
}

func TestIsValidSizing(t *testing.T) {
	for _, tc := range []struct {
		sizing string
		valid  bool
	}{
		{sizing: "size_s", valid: true},
		{sizing: "size_m", valid: true},
		{sizing: "size_l", valid: true},
		{sizing: "", valid: false},
		{sizing: "size_xl", valid: false},
		{sizing: "SIZE_M", valid: false},
		{sizing: " size_m", valid: false},
		{sizing: "medium", valid: false},
	} {
		t.Run(fmt.Sprintf("%q", tc.sizing), func(t *testing.T) {
			assert.Equal(t, tc.valid, IsValidSizing(tc.sizing))
		})
	}
}

// the three preset names are what users write in the odigos configuration and what the UI sends,
// so they are a contract rather than an internal label.
func TestTheSizingNamesUsersSelectAreStable(t *testing.T) {
	assert.Equal(t, "size_s", string(SizeSmall))
	assert.Equal(t, "size_m", string(SizeMedium))
	assert.Equal(t, "size_l", string(SizeLarge))
}

func TestGetResourceSizePresetReturnsTheRequestedPreset(t *testing.T) {
	for _, s := range allSizings {
		t.Run(string(s), func(t *testing.T) {
			assert.Equal(t, configs[s], GetResourceSizePreset(string(s)))
		})
	}
}

func TestGetResourceSizePresetFallsBackToMediumForAnInvalidSizing(t *testing.T) {
	for _, invalid := range []string{"", "size_xl", "SIZE_M", "medium"} {
		t.Run(fmt.Sprintf("%q", invalid), func(t *testing.T) {
			got := GetResourceSizePreset(invalid)

			assert.Equal(t, GetResourceSizePreset(string(SizeMedium)), got)
			assert.NotEqual(t, GetResourceSizePreset(string(SizeSmall)), got)
			assert.NotEqual(t, GetResourceSizePreset(string(SizeLarge)), got)
		})
	}
}

// the preset table is a package level map, so handing out anything that aliases it would let one
// caller's override corrupt the presets for every later reconcile.
func TestGetResourceSizePresetHandsOutACopyOfThePresetTable(t *testing.T) {
	pristine := configs[SizeSmall]

	got := GetResourceSizePreset(string(SizeSmall))
	got.CollectorGatewayConfig.RequestCPUm = 4242
	got.CollectorNodeConfig.RequestCPUm = 4243

	assert.Equal(t, pristine, configs[SizeSmall])
	assert.Equal(t, pristine, GetResourceSizePreset(string(SizeSmall)))
}
