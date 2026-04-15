package otlp

import (
	"context"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

type Profiles struct {
	rx        *Receiver
	component xreceiver.Profiles
	Consumer  xconsumer.Profiles
}

func NewProfilesConsumer(sink xconsumer.Profiles) *Profiles {
	return &Profiles{Consumer: sink}
}

func (p *Profiles) Register(ctx context.Context, rx *Receiver) error {
	if p == nil || p.Consumer == nil {
		return nil
	}
	xf, ok := rx.Factory.(xreceiver.Factory)
	if !ok {
		p.rx = rx
		commonlogger.LoggerCompat().With("subsystem", "ui-otlp", "receiver").Warn(
			"OTLP profiles receiver was not created; profiling ingestion disabled (gateway may log Unimplemented on ProfilesService)")
		return nil
	}
	pr, err := xf.CreateProfiles(ctx, rx.Settings, rx.Cfg, p.Consumer)
	if err != nil {
		return err
	}
	p.rx = rx
	p.component = pr
	rx.profilesReceiver = pr
	return nil
}

func (p *Profiles) Start(ctx context.Context) error {
	if p == nil || p.component == nil {
		return nil
	}
	return p.component.Start(ctx, p.rx.Host)
}

func (p *Profiles) Shutdown(ctx context.Context) error {
	if p == nil || p.component == nil {
		return nil
	}
	return p.component.Shutdown(ctx)
}
