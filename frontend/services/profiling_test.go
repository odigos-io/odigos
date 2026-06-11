package services

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/services/profiles"
	"github.com/stretchr/testify/assert"
)

func TestProfilingRuntimeFromConfig_UIOverlay(t *testing.T) {
	envMax, envTTL, envBytes, _ := profiles.StoreLimitsFromEnv()

	t.Run("nil config falls back to env defaults and ingest off", func(t *testing.T) {
		got := ProfilingRuntimeFromConfig(nil)
		assert.False(t, got.ReceiverOn)
		assert.Equal(t, envMax, got.StoreLimits.MaxSlots)
		assert.Equal(t, envTTL, got.StoreLimits.SlotTTLSeconds)
		assert.Equal(t, envBytes, got.StoreLimits.SlotMaxBytes)
	})

	t.Run("profiling.ui values win over env defaults", func(t *testing.T) {
		enabled := true
		cfg := &common.OdigosConfiguration{
			Profiling: &common.ProfilingConfiguration{
				Enabled: &enabled,
				UI: &common.ProfilingUiConfiguration{
					SlotTTLSeconds: 300,
					MaxSlots:       32,
					SlotMaxBytes:   1234,
				},
			},
		}
		got := ProfilingRuntimeFromConfig(cfg)
		assert.True(t, got.ReceiverOn)
		assert.Equal(t, 300, got.StoreLimits.SlotTTLSeconds)
		assert.Equal(t, 32, got.StoreLimits.MaxSlots)
		assert.Equal(t, 1234, got.StoreLimits.SlotMaxBytes)
	})

	t.Run("zero/unset ui fields keep env defaults", func(t *testing.T) {
		cfg := &common.OdigosConfiguration{
			Profiling: &common.ProfilingConfiguration{
				UI: &common.ProfilingUiConfiguration{SlotTTLSeconds: 300},
			},
		}
		got := ProfilingRuntimeFromConfig(cfg)
		assert.Equal(t, 300, got.StoreLimits.SlotTTLSeconds)
		assert.Equal(t, envMax, got.StoreLimits.MaxSlots)
		assert.Equal(t, envBytes, got.StoreLimits.SlotMaxBytes)
	})
}
