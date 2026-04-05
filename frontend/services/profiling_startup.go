package services

import (
	"context"
	"time"

	"github.com/odigos-io/odigos/common"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	collectorprofiles "github.com/odigos-io/odigos/frontend/services/collector_profiles"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveProfilingFromEffectiveConfig decides whether the UI pod should listen for OTLP profiles (gRPC),
// and the in-memory store limits (slots, TTL, bytes per slot, cleanup ticker).
//
// Flow:
//  1. Baseline limits from StoreConfigFromEnv: defaults match defaults.go / Helm profiling.ui; optional
//     PROFILES_* env vars override on the pod (local dev / ops) before config is applied.
//  2. Effective Odigos configuration from the cluster (GetEffectiveConfig): when present, profiling.ui
//     overrides maxSlots, slot TTL, and slotMaxBytes via applyProfilingUiOverrides (same keys as Helm
//     writes into odigos-configuration).
//  3. If effective config does not enable profiling, fall back to Helm deployment config
//     (GetHelmDeploymentConfig) so the UI can still receive profiles before the effective ConfigMap catches up.
//  4. receiverOn is true only when the chosen config (effective or Helm fallback) has profiling enabled
//     per ProfilingPipelineActive / ProfilingEnabled (explicit enabled: true).
//
// Env vars remain useful for tests and operators who do not patch odigos-configuration; they are not
// required when Helm + effective config define profiling.ui.
func ResolveProfilingFromEffectiveConfig(ctx context.Context, c client.Client) (receiverOn bool, otlpGrpcPort int, maxSlots, ttlSec, slotMaxBytes int, cleanupInterval time.Duration, err error) {
	maxSlots, ttlSec, slotMaxBytes, cleanup := collectorprofiles.StoreConfigFromEnv()
	otlpGrpcPort = odigosconsts.OTLPPort

	cfg, err := GetEffectiveConfig(ctx, c)
	if err != nil {
		return false, otlpGrpcPort, maxSlots, ttlSec, slotMaxBytes, cleanup, err
	}

	var useCfg *common.OdigosConfiguration
	if cfg != nil {
		// If the live effective config explicitly sets profiling.enabled to false, turn the receiver off
		// and do not fall back to Helm—otherwise Helm could keep profiles on while the user disabled profiling.
		if cfg.Profiling != nil && cfg.Profiling.Enabled != nil && !*cfg.Profiling.Enabled {
			return false, otlpGrpcPort, maxSlots, ttlSec, slotMaxBytes, cleanup, nil
		}
		if cfg.ProfilingEnabled() {
			useCfg = cfg
		}
	}
	if useCfg == nil {
		helmCfg, helmErr := GetHelmDeploymentConfig(ctx, c)
		if helmErr != nil {
			return false, otlpGrpcPort, maxSlots, ttlSec, slotMaxBytes, cleanup, helmErr
		}
		if helmCfg != nil && helmCfg.ProfilingEnabled() {
			useCfg = helmCfg
		}
	}

	applyProfilingUiOverrides(useCfg, &maxSlots, &ttlSec, &slotMaxBytes)
	receiverOn = useCfg != nil
	return receiverOn, otlpGrpcPort, maxSlots, ttlSec, slotMaxBytes, cleanup, nil
}

func applyProfilingUiOverrides(cfg *common.OdigosConfiguration, maxSlots, ttlSec, slotMaxBytes *int) {
	if cfg == nil || cfg.Profiling == nil || cfg.Profiling.Ui == nil {
		return
	}
	u := cfg.Profiling.Ui
	if u.MaxSlots > 0 {
		*maxSlots = u.MaxSlots
	}
	if u.SlotTTLSeconds > 0 {
		*ttlSec = u.SlotTTLSeconds
	}
	if u.SlotMaxBytes > 0 {
		*slotMaxBytes = u.SlotMaxBytes
	}
}
