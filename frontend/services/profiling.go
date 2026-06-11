package services

import (
	"context"
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/services/profiles"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProfilingStoreLimits holds resolved limits for the in-memory profile store.
type ProfilingStoreLimits struct {
	MaxSlots       int
	SlotTTLSeconds int
	SlotMaxBytes   int
}

// ProfilingRuntimeConfig is the UI process decision for OTLP profiling ingest and store sizing.
type ProfilingRuntimeConfig struct {
	ReceiverOn      bool
	StoreLimits     ProfilingStoreLimits
	CleanupInterval time.Duration // ProfileStore TTL sweep period
}

func ResolveProfilingFromEffectiveConfig(ctx context.Context, c client.Client) (ProfilingRuntimeConfig, error) {
	out := profilingRuntimeFromConfig(nil)

	cfg, err := GetEffectiveConfig(ctx, c)
	if err != nil {
		return out, err
	}
	return profilingRuntimeFromConfig(cfg), nil
}

// profilingRuntimeFromConfig resolves the UI profile store sizing for a given effective-config snapshot.
// Store limits default to the pod env (PROFILES_*), then any profiling.ui values set in the config win.
func profilingRuntimeFromConfig(cfg *common.OdigosConfiguration) ProfilingRuntimeConfig {
	maxSlots, ttlSec, slotMaxBytes, cleanup := profiles.StoreLimitsFromEnv()

	if cfg != nil && cfg.Profiling != nil && cfg.Profiling.UI != nil {
		ui := cfg.Profiling.UI
		if ui.MaxSlots > 0 {
			maxSlots = ui.MaxSlots
		}
		if ui.SlotTTLSeconds > 0 {
			ttlSec = ui.SlotTTLSeconds
		}
		if ui.SlotMaxBytes > 0 {
			slotMaxBytes = ui.SlotMaxBytes
		}
	}

	return ProfilingRuntimeConfig{
		ReceiverOn: ProfilingEnabledFromOdigosConfig(cfg),
		StoreLimits: ProfilingStoreLimits{
			MaxSlots:       maxSlots,
			SlotTTLSeconds: ttlSec,
			SlotMaxBytes:   slotMaxBytes,
		},
		CleanupInterval: cleanup,
	}
}

// ProfilingRuntimeFromConfig resolves UI profile store sizing and ingest state for an effective-config snapshot.
func ProfilingRuntimeFromConfig(cfg *common.OdigosConfiguration) ProfilingRuntimeConfig {
	return profilingRuntimeFromConfig(cfg)
}

// ProfilingEnabledFromOdigosConfig reports whether the UI should accept OTLP profiles for this config snapshot.
func ProfilingEnabledFromOdigosConfig(cfg *common.OdigosConfiguration) bool {
	return cfg != nil && cfg.ProfilingEnabled()
}
