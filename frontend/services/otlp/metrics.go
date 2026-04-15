package otlp

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	recv "go.opentelemetry.io/collector/receiver"
)

type Metrics struct {
	rx        *Receiver
	component recv.Metrics
	Consumer  consumer.Metrics
}

func NewMetrics(sink consumer.Metrics) *Metrics {
	return &Metrics{Consumer: sink}
}

func (m *Metrics) Register(ctx context.Context, rx *Receiver) error {
	if m.Consumer == nil {
		return fmt.Errorf("otlp metrics: sink is nil")
	}

	c, err := rx.Factory.CreateMetrics(ctx, rx.Settings, rx.Cfg, m.Consumer)
	if err != nil {
		return err
	}
	m.rx = rx
	m.component = c
	rx.metricsReceiver = c
	return nil
}

func (m *Metrics) Start(ctx context.Context) error {
	if m.rx == nil || m.component == nil {
		return fmt.Errorf("otlp metrics: Register first")
	}
	return m.component.Start(ctx, m.rx.Host)
}

func (m *Metrics) Shutdown(ctx context.Context) error {
	if m.component == nil {
		return nil
	}
	return m.component.Shutdown(ctx)
}
