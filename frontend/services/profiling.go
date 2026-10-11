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
	maxSlots, ttlSec, slotMaxBytes, cleanup := profiles.StoreLimitsFromEnv()
	out := ProfilingRuntimeConfig{
		StoreLimits: ProfilingStoreLimits{
			MaxSlots:       maxSlots,
			SlotTTLSeconds: ttlSec,
			SlotMaxBytes:   slotMaxBytes,
		},
		CleanupInterval: cleanup,
	}

	cfg, err := GetEffectiveConfig(ctx, c)
	if err != nil {
		return out, err
	}

	if ProfilingEnabledFromOdigosConfig(cfg) {
		out.ReceiverOn = true
	}
	// Config-provided cache limits (profiling.ui) override the env defaults, so
	// the settings page can tune them live through effective-config.
	if cfg != nil && cfg.Profiling != nil && cfg.Profiling.Ui != nil {
		ui := cfg.Profiling.Ui
		if ui.MaxSlots > 0 {
			out.StoreLimits.MaxSlots = ui.MaxSlots
		}
		if ui.SlotMaxBytes > 0 {
			out.StoreLimits.SlotMaxBytes = ui.SlotMaxBytes
		}
		if ui.SlotTTLSeconds > 0 {
			out.StoreLimits.SlotTTLSeconds = ui.SlotTTLSeconds
		}
	}
	return out, nil
}

// ProfilingEnabledFromOdigosConfig reports whether the UI should accept OTLP profiles for this config snapshot.
func ProfilingEnabledFromOdigosConfig(cfg *common.OdigosConfiguration) bool {
	return cfg != nil && cfg.ProfilingEnabled()
}
