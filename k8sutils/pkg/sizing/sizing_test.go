package sizing

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }
func labelsPtr(l map[string]string) *map[string]string {
	return &l
}

// distinct sentinel values, one per sizing field, so that a field copied from or written to the
// wrong neighbour shows up as a wrong number instead of an identical one.
func sentinelGatewaySizing() common.CollectorGatewayConfiguration {
	return common.CollectorGatewayConfiguration{
		MinReplicas:                7001,
		MaxReplicas:                7002,
		RequestMemoryMiB:           7003,
		LimitMemoryMiB:             7004,
		RequestCPUm:                7005,
		LimitCPUm:                  7006,
		MemoryLimiterLimitMiB:      7007,
		MemoryLimiterSpikeLimitMiB: 7008,
		GoMemLimitMib:              7009,
	}
}

func sentinelNodeSizing() common.CollectorNodeConfiguration {
	return common.CollectorNodeConfiguration{
		RequestMemoryMiB:           8001,
		LimitMemoryMiB:             8002,
		RequestCPUm:                8003,
		LimitCPUm:                  8004,
		MemoryLimiterLimitMiB:      8005,
		MemoryLimiterSpikeLimitMiB: 8006,
		GoMemLimitMib:              8007,
	}
}

// a gateway configuration with every field populated: the sizing fields hold values that differ
// from both the presets and the sentinels above, and every non-sizing field is set so that
// dropping it is observable.
func userGatewayConfig() common.CollectorGatewayConfiguration {
	return common.CollectorGatewayConfiguration{
		MinReplicas:                9001,
		MaxReplicas:                9002,
		RequestMemoryMiB:           9003,
		LimitMemoryMiB:             9004,
		RequestCPUm:                9005,
		LimitCPUm:                  9006,
		MemoryLimiterLimitMiB:      9007,
		MemoryLimiterSpikeLimitMiB: 9008,
		GoMemLimitMib:              9009,
		ServiceGraph: &common.ServiceGraphOptions{
			Disabled:        boolPtr(true),
			ExtraDimensions: []string{"http.route"},
		},
		ClusterMetricsEnabled: boolPtr(true),
		HttpsProxyAddress:     strPtr("http://proxy.odigos-system:3128"),
		NodeSelector:          labelsPtr(map[string]string{"odigos.io/gateway": "true"}),
		DeploymentName:        "my-own-gateway",
	}
}

func userNodeConfig() common.CollectorNodeConfiguration {
	return common.CollectorNodeConfiguration{
		CollectorOwnMetricsPort:    55555,
		RequestMemoryMiB:           9101,
		LimitMemoryMiB:             9102,
		RequestCPUm:                9103,
		LimitCPUm:                  9104,
		MemoryLimiterLimitMiB:      9105,
		MemoryLimiterSpikeLimitMiB: 9106,
		GoMemLimitMib:              9107,
		EnableDataCompression:      boolPtr(true),
		OtlpExporterConfiguration: &common.OtlpExporterConfiguration{
			Timeout: "17s",
		},
		ResourceDetectors: &common.ResourceDetectorsConfiguration{
			EKS: &common.ResourceDetectorConfig{},
		},
	}
}

func assertFieldsEqual(t *testing.T, names []string, want any, got any) {
	t.Helper()
	wantValue := reflect.ValueOf(want)
	gotValue := reflect.ValueOf(got)
	for _, name := range names {
		assert.Equal(t, wantValue.FieldByName(name).Interface(), gotValue.FieldByName(name).Interface(),
			"field %q", name)
	}
}

// setOneIntField returns a copy of cfg with exactly one int field set to value.
func setOneIntField[T any](cfg T, name string, value int) T {
	field := reflect.ValueOf(&cfg).Elem().FieldByName(name)
	field.SetInt(int64(value))
	return cfg
}

func TestCopyNonZeroGatewayOverlaysExactlyTheFieldThatWasSet(t *testing.T) {
	const override = 6543

	for _, name := range gatewaySizingFields {
		t.Run(name, func(t *testing.T) {
			base := sentinelGatewaySizing()
			src := setOneIntField(common.CollectorGatewayConfiguration{}, name, override)

			got := copyNonZeroGateway(&base, &src)

			assert.Equal(t, override, intFieldOf(t, *got, name), "the overridden field was not applied")
			for _, other := range gatewaySizingFields {
				if other == name {
					continue
				}
				assert.Equal(t, intFieldOf(t, sentinelGatewaySizing(), other), intFieldOf(t, *got, other),
					"overriding %q also changed %q", name, other)
			}
		})
	}
}

func TestCopyNonZeroNodeOverlaysExactlyTheFieldThatWasSet(t *testing.T) {
	const override = 6544

	for _, name := range nodeSizingFields {
		t.Run(name, func(t *testing.T) {
			src := setOneIntField(common.CollectorNodeConfiguration{}, name, override)

			got := copyNonZeroNode(sentinelNodeSizing(), &src)

			assert.Equal(t, override, intFieldOf(t, got, name), "the overridden field was not applied")
			for _, other := range nodeSizingFields {
				if other == name {
					continue
				}
				assert.Equal(t, intFieldOf(t, sentinelNodeSizing(), other), intFieldOf(t, got, other),
					"overriding %q also changed %q", name, other)
			}
		})
	}
}

func TestCopyNonZeroKeepsTheBaseWhenThereIsNoOverride(t *testing.T) {
	t.Run("gateway/nil", func(t *testing.T) {
		base := sentinelGatewaySizing()
		assert.Equal(t, sentinelGatewaySizing(), *copyNonZeroGateway(&base, nil))
	})

	t.Run("gateway/all zero", func(t *testing.T) {
		base := sentinelGatewaySizing()
		assert.Equal(t, sentinelGatewaySizing(), *copyNonZeroGateway(&base, &common.CollectorGatewayConfiguration{}))
	})

	t.Run("node/nil", func(t *testing.T) {
		assert.Equal(t, sentinelNodeSizing(), copyNonZeroNode(sentinelNodeSizing(), nil))
	})

	t.Run("node/all zero", func(t *testing.T) {
		assert.Equal(t, sentinelNodeSizing(), copyNonZeroNode(sentinelNodeSizing(), &common.CollectorNodeConfiguration{}))
	})
}

// the overlay only knows about sizing, so a user setting that is not sizing must not survive it.
// this is exactly why ComputeEffectiveCollectorConfig merges on top of the user configuration
// instead of returning the overlay result directly.
func TestCopyNonZeroDropsNonSizingConfiguration(t *testing.T) {
	t.Run("gateway", func(t *testing.T) {
		base := sentinelGatewaySizing()
		user := userGatewayConfig()

		got := copyNonZeroGateway(&base, &user)

		for _, name := range gatewayPreservedFields {
			field := reflect.ValueOf(*got).FieldByName(name)
			require.True(t, field.IsValid())
			assert.True(t, field.IsZero(), "field %q unexpectedly survived the sizing overlay", name)
		}
	})

	t.Run("node", func(t *testing.T) {
		user := userNodeConfig()

		got := copyNonZeroNode(sentinelNodeSizing(), &user)

		for _, name := range nodePreservedFields {
			field := reflect.ValueOf(got).FieldByName(name)
			require.True(t, field.IsValid())
			assert.True(t, field.IsZero(), "field %q unexpectedly survived the sizing overlay", name)
		}
	})
}

func TestMergeGatewayConfigurationTakesSizingFromThePresetAndKeepsEverythingElse(t *testing.T) {
	existing := userGatewayConfig()
	preset := sentinelGatewaySizing()

	got := mergeGatewayConfiguration(&existing, &preset)

	require.NotNil(t, got)
	assertFieldsEqual(t, gatewaySizingFields, sentinelGatewaySizing(), *got)
	assertFieldsEqual(t, gatewayPreservedFields, userGatewayConfig(), *got)
	assert.Equal(t, userGatewayConfig(), existing, "the caller's configuration must not be modified")
}

func TestMergeNodeConfigurationTakesSizingFromThePresetAndKeepsEverythingElse(t *testing.T) {
	existing := userNodeConfig()

	got := mergeNodeConfiguration(&existing, sentinelNodeSizing())

	require.NotNil(t, got)
	assertFieldsEqual(t, nodeSizingFields, sentinelNodeSizing(), *got)
	assertFieldsEqual(t, nodePreservedFields, userNodeConfig(), *got)
	assert.Equal(t, userNodeConfig(), existing, "the caller's configuration must not be modified")
}

func TestMergeConfigurationFallsBackToThePresetWhenThereIsNoExistingConfiguration(t *testing.T) {
	t.Run("gateway", func(t *testing.T) {
		preset := sentinelGatewaySizing()
		assert.Equal(t, sentinelGatewaySizing(), *mergeGatewayConfiguration(nil, &preset))
	})

	t.Run("node", func(t *testing.T) {
		assert.Equal(t, sentinelNodeSizing(), *mergeNodeConfiguration(nil, sentinelNodeSizing()))
	})
}

func TestComputeResourceSizePresetNormalizesAnInvalidPresetOnTheConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		selected string
		want     string
	}{
		{name: "empty", selected: "", want: "size_m"},
		{name: "unknown", selected: "size_xl", want: "size_m"},
		{name: "valid is left alone", selected: "size_l", want: "size_l"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &common.OdigosConfiguration{ResourceSizePreset: tc.selected}

			ComputeResourceSizePreset(config)

			// the scheduler persists this configuration as the effective config that the UI reads
			// back, so the normalized value has to land on the caller's object.
			assert.Equal(t, tc.want, config.ResourceSizePreset)
		})
	}
}

func TestComputeResourceSizePresetUsesTheSelectedPresetWhenThereAreNoOverrides(t *testing.T) {
	for _, s := range allSizings {
		t.Run(string(s), func(t *testing.T) {
			got := ComputeResourceSizePreset(&common.OdigosConfiguration{ResourceSizePreset: string(s)})

			assert.Equal(t, GetResourceSizePreset(string(s)), got)
		})
	}
}

func TestComputeResourceSizePresetLetsTheUserOverrideExactlyOneField(t *testing.T) {
	const override = 4711
	preset := GetResourceSizePreset(string(SizeSmall))

	for _, name := range gatewaySizingFields {
		t.Run("gateway/"+name, func(t *testing.T) {
			gwOverride := setOneIntField(common.CollectorGatewayConfiguration{}, name, override)

			got := ComputeResourceSizePreset(&common.OdigosConfiguration{
				ResourceSizePreset: string(SizeSmall),
				CollectorGateway:   &gwOverride,
			})

			assert.Equal(t, override, intFieldOf(t, got.CollectorGatewayConfig, name))
			for _, other := range gatewaySizingFields {
				if other == name {
					continue
				}
				assert.Equal(t, intFieldOf(t, preset.CollectorGatewayConfig, other),
					intFieldOf(t, got.CollectorGatewayConfig, other), "overriding %q also changed %q", name, other)
			}
			assert.Equal(t, preset.CollectorNodeConfig, got.CollectorNodeConfig,
				"a gateway override must not touch the node collector configuration")
		})
	}

	for _, name := range nodeSizingFields {
		t.Run("node/"+name, func(t *testing.T) {
			nodeOverride := setOneIntField(common.CollectorNodeConfiguration{}, name, override)

			got := ComputeResourceSizePreset(&common.OdigosConfiguration{
				ResourceSizePreset: string(SizeSmall),
				CollectorNode:      &nodeOverride,
			})

			assert.Equal(t, override, intFieldOf(t, got.CollectorNodeConfig, name))
			for _, other := range nodeSizingFields {
				if other == name {
					continue
				}
				assert.Equal(t, intFieldOf(t, preset.CollectorNodeConfig, other),
					intFieldOf(t, got.CollectorNodeConfig, other), "overriding %q also changed %q", name, other)
			}
			assert.Equal(t, preset.CollectorGatewayConfig, got.CollectorGatewayConfig,
				"a node override must not touch the gateway configuration")
		})
	}
}

// the presets live in a package level map and the overlay writes into its own copy of the entry.
// if that copy ever stops being a copy, the first user override would rewrite the shipped preset
// for every later call in the process.
func TestAUserOverrideDoesNotLeakIntoThePresetTable(t *testing.T) {
	pristine := GetResourceSizePreset(string(SizeSmall))

	overridden := ComputeResourceSizePreset(&common.OdigosConfiguration{
		ResourceSizePreset: string(SizeSmall),
		CollectorGateway:   &common.CollectorGatewayConfiguration{RequestCPUm: 31337},
		CollectorNode:      &common.CollectorNodeConfiguration{RequestCPUm: 31338},
	})
	require.Equal(t, 31337, overridden.CollectorGatewayConfig.RequestCPUm)
	require.Equal(t, 31338, overridden.CollectorNodeConfig.RequestCPUm)

	assert.Equal(t, pristine, ComputeResourceSizePreset(&common.OdigosConfiguration{
		ResourceSizePreset: string(SizeSmall),
	}))
	assert.Equal(t, pristine, MergeSizing(string(SizeSmall), nil, nil))
}

func TestMergeSizingMatchesComputeResourceSizePreset(t *testing.T) {
	for _, preset := range []string{"size_s", "size_m", "size_l", "size_xl", ""} {
		for _, tc := range []struct {
			name string
			gw   *common.CollectorGatewayConfiguration
			node *common.CollectorNodeConfiguration
		}{
			{name: "no overrides"},
			{name: "gateway only", gw: &common.CollectorGatewayConfiguration{MaxReplicas: 42}},
			{name: "node only", node: &common.CollectorNodeConfiguration{LimitCPUm: 4242}},
			{
				name: "both",
				gw:   &common.CollectorGatewayConfiguration{MinReplicas: 4, GoMemLimitMib: 111},
				node: &common.CollectorNodeConfiguration{RequestMemoryMiB: 222},
			},
		} {
			t.Run(fmt.Sprintf("%q/%s", preset, tc.name), func(t *testing.T) {
				viaConfig := ComputeResourceSizePreset(&common.OdigosConfiguration{
					ResourceSizePreset: preset,
					CollectorGateway:   tc.gw,
					CollectorNode:      tc.node,
				})

				assert.Equal(t, viaConfig, MergeSizing(preset, tc.gw, tc.node))
			})
		}
	}
}

func TestComputeEffectiveCollectorConfigAppliesThePresetToAConfigWithoutSizing(t *testing.T) {
	for _, s := range allSizings {
		t.Run(string(s), func(t *testing.T) {
			gateway := userGatewayConfig()
			node := userNodeConfig()
			// only the non-sizing fields are user provided here, so every sizing field must be
			// filled in from the preset.
			for _, name := range gatewaySizingFields {
				gateway = setOneIntField(gateway, name, 0)
			}
			for _, name := range nodeSizingFields {
				node = setOneIntField(node, name, 0)
			}

			effectiveGateway, effectiveNode := ComputeEffectiveCollectorConfig(&common.OdigosConfiguration{
				ResourceSizePreset: string(s),
				CollectorGateway:   &gateway,
				CollectorNode:      &node,
			})

			require.NotNil(t, effectiveGateway)
			require.NotNil(t, effectiveNode)
			preset := GetResourceSizePreset(string(s))
			assertFieldsEqual(t, gatewaySizingFields, preset.CollectorGatewayConfig, *effectiveGateway)
			assertFieldsEqual(t, nodeSizingFields, preset.CollectorNodeConfig, *effectiveNode)
			assertFieldsEqual(t, gatewayPreservedFields, userGatewayConfig(), *effectiveGateway)
			assertFieldsEqual(t, nodePreservedFields, userNodeConfig(), *effectiveNode)
		})
	}
}

func TestComputeEffectiveCollectorConfigKeepsUserSizingAndEverythingElse(t *testing.T) {
	gateway := userGatewayConfig()
	node := userNodeConfig()

	effectiveGateway, effectiveNode := ComputeEffectiveCollectorConfig(&common.OdigosConfiguration{
		ResourceSizePreset: string(SizeLarge),
		CollectorGateway:   &gateway,
		CollectorNode:      &node,
	})

	require.NotNil(t, effectiveGateway)
	require.NotNil(t, effectiveNode)
	assert.Equal(t, userGatewayConfig(), *effectiveGateway, "an explicit user value wins over the preset")
	assert.Equal(t, userNodeConfig(), *effectiveNode, "an explicit user value wins over the preset")
	assert.Equal(t, userGatewayConfig(), gateway, "the caller's configuration must not be modified")
	assert.Equal(t, userNodeConfig(), node, "the caller's configuration must not be modified")
}

func TestComputeEffectiveCollectorConfigFillsOnlyTheSizingFieldsTheUserLeftEmpty(t *testing.T) {
	const override = 1234
	preset := GetResourceSizePreset(string(SizeMedium))

	for _, name := range gatewaySizingFields {
		t.Run("gateway/"+name, func(t *testing.T) {
			gateway := setOneIntField(common.CollectorGatewayConfiguration{}, name, override)

			effectiveGateway, _ := ComputeEffectiveCollectorConfig(&common.OdigosConfiguration{
				ResourceSizePreset: string(SizeMedium),
				CollectorGateway:   &gateway,
			})

			require.NotNil(t, effectiveGateway)
			assert.Equal(t, override, intFieldOf(t, *effectiveGateway, name))
			for _, other := range gatewaySizingFields {
				if other == name {
					continue
				}
				assert.Equal(t, intFieldOf(t, preset.CollectorGatewayConfig, other),
					intFieldOf(t, *effectiveGateway, other), "setting %q also changed %q", name, other)
			}
		})
	}

	for _, name := range nodeSizingFields {
		t.Run("node/"+name, func(t *testing.T) {
			node := setOneIntField(common.CollectorNodeConfiguration{}, name, override)

			_, effectiveNode := ComputeEffectiveCollectorConfig(&common.OdigosConfiguration{
				ResourceSizePreset: string(SizeMedium),
				CollectorNode:      &node,
			})

			require.NotNil(t, effectiveNode)
			assert.Equal(t, override, intFieldOf(t, *effectiveNode, name))
			for _, other := range nodeSizingFields {
				if other == name {
					continue
				}
				assert.Equal(t, intFieldOf(t, preset.CollectorNodeConfig, other),
					intFieldOf(t, *effectiveNode, other), "setting %q also changed %q", name, other)
			}
		})
	}
}

func TestComputeEffectiveCollectorConfigReturnsThePresetForAConfigWithoutCollectorSections(t *testing.T) {
	effectiveGateway, effectiveNode := ComputeEffectiveCollectorConfig(&common.OdigosConfiguration{
		ResourceSizePreset: string(SizeLarge),
	})

	require.NotNil(t, effectiveGateway)
	require.NotNil(t, effectiveNode)
	preset := GetResourceSizePreset(string(SizeLarge))
	assert.Equal(t, preset.CollectorGatewayConfig, *effectiveGateway)
	assert.Equal(t, preset.CollectorNodeConfig, *effectiveNode)
}

func TestComputeEffectiveCollectorConfigFallsBackToMediumForAnInvalidPreset(t *testing.T) {
	effectiveGateway, effectiveNode := ComputeEffectiveCollectorConfig(&common.OdigosConfiguration{
		ResourceSizePreset: "size_xl",
	})

	require.NotNil(t, effectiveGateway)
	require.NotNil(t, effectiveNode)
	medium := GetResourceSizePreset(string(SizeMedium))
	assert.Equal(t, medium.CollectorGatewayConfig, *effectiveGateway)
	assert.Equal(t, medium.CollectorNodeConfig, *effectiveNode)
}

// two reconciles of the same configuration have to produce the same effective configuration,
// otherwise the scheduler rewrites the effective config map and restarts the collectors forever.
func TestComputeEffectiveCollectorConfigIsStableAcrossRepeatedCalls(t *testing.T) {
	config := common.OdigosConfiguration{
		ResourceSizePreset: "size_xl",
		CollectorGateway:   &common.CollectorGatewayConfiguration{MaxReplicas: 20, DeploymentName: "gw"},
		CollectorNode:      &common.CollectorNodeConfiguration{CollectorOwnMetricsPort: 1234},
	}

	firstGateway, firstNode := ComputeEffectiveCollectorConfig(&config)
	config.CollectorGateway = firstGateway
	config.CollectorNode = firstNode
	secondGateway, secondNode := ComputeEffectiveCollectorConfig(&config)

	assert.Equal(t, *firstGateway, *secondGateway)
	assert.Equal(t, *firstNode, *secondNode)
}
