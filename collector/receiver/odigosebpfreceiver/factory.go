package odigosebpfreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

//go:generate mdatagen metadata.yaml

const TypeStr = "odigosebpf"

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(TypeStr),
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, component.StabilityLevelBeta),
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelBeta),
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelBeta),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		MetricsConfig: MetricsConfig{
			Interval: 30 * time.Second, // Default to 30 seconds
		},
	}
}

func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	next consumer.Traces,
) (receiver.Traces, error) {
	return &ebpfReceiver{
		config:       cfg.(*Config),
		receiverType: ReceiverTypeTraces,
		nextTraces:   next,
		logger:       set.Logger,
		settings:     set,
	}, nil
}

func createMetricsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	next consumer.Metrics,
) (receiver.Metrics, error) {
	return &ebpfReceiver{
		config:       cfg.(*Config),
		receiverType: ReceiverTypeMetrics,
		nextMetrics:  next,
		logger:       set.Logger,
		settings:     set,
	}, nil
}

func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	next consumer.Logs,
) (receiver.Logs, error) {
	return &ebpfReceiver{
		config:       cfg.(*Config),
		receiverType: ReceiverTypeLogs,
		nextLogs:     next,
		logger:       set.Logger,
		settings:     set,
	}, nil
}
