package otlp

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

type Profiles struct {
	otlpReceiver *Receiver
	receiver     xreceiver.Profiles
	consumer     xconsumer.Profiles
}

var _ OTLPPipeline = (*Profiles)(nil)

func NewProfilesPipeline(r *Receiver, c xconsumer.Profiles) *Profiles {
	return &Profiles{otlpReceiver: r, consumer: c}
}

func (p *Profiles) Register(ctx context.Context) error {
	xf, ok := p.otlpReceiver.Factory.(xreceiver.Factory)
	if !ok {
		return fmt.Errorf("otlp: receiver factory does not support profiles (expected xreceiver.Factory)")
	}
	profilesReceiver, err := xf.CreateProfiles(ctx, p.otlpReceiver.Settings, p.otlpReceiver.Cfg, p.consumer)
	if err != nil {
		return err
	}
	p.receiver = profilesReceiver
	return nil
}

func (p *Profiles) Start(ctx context.Context) error {
	if p.otlpReceiver == nil || p.otlpReceiver.Host == nil {
		return fmt.Errorf("otlp: profiles start requires non-nil host")
	}
	return p.receiver.Start(ctx, p.otlpReceiver.Host)
}

func (p *Profiles) Shutdown(ctx context.Context) error {
	return p.receiver.Shutdown(ctx)
}
