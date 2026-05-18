package odigosprofilesprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/collector"
)

// odigosProfilesProcessor keeps profile batches only for workloads that appear in the
// configured OdigosConfigExtension cache.
//
// Must run after k8s_attributes (or equivalent enrichment) on the profiles pipeline so
// namespace/kind/name are present on the resource.
type odigosProfilesProcessor struct {
	logger *zap.Logger
	cfg    *Config

	provider        collector.OdigosConfigExtension
	droppedProfiles metric.Int64Counter
}

func newOdigosProfilesProcessor(logger *zap.Logger, tel component.TelemetrySettings, cfg *Config) *odigosProfilesProcessor {
	meter := tel.MeterProvider.Meter("github.com/odigos-io/odigos/collector/processors/odigosprofilesprocessor")
	dropped, _ := meter.Int64Counter(
		"odigos.profiles.processor.dropped.resource_profiles",
		metric.WithDescription("ResourceProfiles dropped because the workload is not in the odigos_config_k8s cache or workload identity could not be derived from resource attributes"),
	)
	return &odigosProfilesProcessor{
		logger:          logger,
		cfg:             cfg,
		droppedProfiles: dropped,
	}
}

func (p *odigosProfilesProcessor) Start(ctx context.Context, host component.Host) error {
	extID := p.cfg.OdigosConfigExtension
	ext, ok := host.GetExtensions()[*extID]
	if !ok {
		return fmt.Errorf("odigos config extension %q not found", extID.String())
	}
	odigosExt, ok := ext.(collector.OdigosConfigExtension)
	if !ok {
		return fmt.Errorf("extension %q is not an OdigosConfigExtension (got %T)", extID.String(), ext)
	}
	p.provider = odigosExt
	if !p.provider.WaitForCacheSync(ctx) {
		p.logger.Warn("odigosprofilesprocessor: odigos config extension cache sync did not complete; workload cache may be incomplete briefly")
	}
	return nil
}

func (p *odigosProfilesProcessor) Shutdown(context.Context) error {
	p.provider = nil
	return nil
}

func (p *odigosProfilesProcessor) processProfiles(ctx context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
	pd.ResourceProfiles().RemoveIf(func(rp pprofile.ResourceProfiles) bool {
		if p.provider.HasCachedWorkloadContainerConfig(rp.Resource()) {
			return false
		}
		p.droppedProfiles.Add(ctx, 1)
		return true
	})
	return pd, nil
}
