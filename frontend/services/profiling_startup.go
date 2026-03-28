package services

import (
	"context"
	"time"

	"github.com/odigos-io/odigos/common"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	collectorprofiles "github.com/odigos-io/odigos/frontend/services/collector_profiles"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveProfilingFromEffectiveConfig loads profiling receiver enablement, OTLP gRPC port, and store limits.
// It prefers the effective-config ConfigMap (scheduler-resolved). If that is missing or profiling is not
// enabled there yet, it falls back to odigos-configuration (Helm values) so the UI can register the
// OTLP Profiles gRPC service on startup even when effective-config hasn't been reconciled or the cache
// is not ready — otherwise the cluster gateway's profile exporter hits Unimplemented on ui:4317.
//
// Production (e.g. EKS) installs usually use one release tag for all Odigos images, so odigos-ui and
// odigos-gateway share the same OpenTelemetry Collector / OTLP Profiles gRPC version. Dev clusters
// (e.g. kind) often mix a locally built or older ui image with a newer gateway — that mismatch surfaces
// as Unimplemented on ProfilesService or zero odigos_ui_profiling_resource_profiles_stored_total until
// images are rebuilt together from this repo.
func ResolveProfilingFromEffectiveConfig(ctx context.Context, c client.Client) (receiverOn bool, otlpGrpcPort int, maxSlots, ttlSec, slotMaxBytes int, cleanupInterval time.Duration, err error) {
	maxSlots, ttlSec, slotMaxBytes, cleanup := collectorprofiles.StoreConfigFromEnv()
	otlpGrpcPort = odigosconsts.OTLPPort

	cfg, err := GetEffectiveConfig(ctx, c)
	if err != nil {
		return false, otlpGrpcPort, maxSlots, ttlSec, slotMaxBytes, cleanup, err
	}

	var useCfg *common.OdigosConfiguration
	if cfg != nil {
		// Respect an explicit "off" in effective config; do not override with Helm.
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

	if useCfg == nil {
		return false, otlpGrpcPort, maxSlots, ttlSec, slotMaxBytes, cleanup, nil
	}

	applyProfilingUiOverrides(useCfg, &maxSlots, &ttlSec, &slotMaxBytes)
	return true, otlpGrpcPort, maxSlots, ttlSec, slotMaxBytes, cleanup, nil
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
