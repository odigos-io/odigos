package odigosebpfreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const TypeStr = "odigosebpf"

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType(TypeStr),
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, component.StabilityLevelBeta),
	)
}

func createDefaultConfig() component.Config {
	// Return default config object – no extra parameters yet
	return &Config{
		Settings:               receiver.Settings{},
		MaxReadBatchSize:       1024,
		PollInterval:           time.Millisecond * 5,
		MaxGoroutinesPerBuffer: 1,
	}
}

func createTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	next consumer.Traces,
) (receiver.Traces, error) {
	tracesEbpfMapPath := "/sys/fs/bpf/odiglet/traces"
	return &ebpfReceiver{config: cfg.(*Config), nextTraces: next, logger: set.Logger, mapPath: tracesEbpfMapPath}, nil
}
