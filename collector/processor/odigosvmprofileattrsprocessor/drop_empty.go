package odigosvmprofileattrsprocessor

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

// dropEmptyProfilesConsumer is the terminal consumer in the processor chain. It forwards only
// exportable profile batches to the next consumer (exporter) and silently drops empty payloads
// so downstream OTLP backends are not called with zero resource profiles. Per-ResourceProfile
// drops are already counted in vmProfileAttrsProcessor.processProfiles; this consumer does not
// emit telemetry to avoid double-counting batches that contained already-counted drops.
type dropEmptyProfilesConsumer struct {
	next   xconsumer.Profiles
	logger *zap.Logger
}

// Capabilities reports that this wrapper does not mutate data; the upstream processor declares
// its own MutatesData=true capability.
func (d *dropEmptyProfilesConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeProfiles drops empty payloads and forwards everything else.
func (d *dropEmptyProfilesConsumer) ConsumeProfiles(ctx context.Context, md pprofile.Profiles) error {
	if !profilesExportable(md) {
		if d.logger != nil {
			d.logger.Debug("skipping profiles export: no resource profiles after filtering")
		}
		return nil
	}
	return d.next.ConsumeProfiles(ctx, md)
}
