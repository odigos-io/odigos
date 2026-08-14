package odigoscapabilitiesextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/capabilities"
)

// capabilitiesExtension raises the process effective capability set to match the
// permitted set on startup, before pipeline components (e.g. the eBPF receiver)
// start.
//
// This is necessary when the collector is run as a non-root user with elevated capabilities (e.g. via `setcap`).
// since we are using the same image, and same binary for both the node collector and the cluster collector,
// the node collector needs eBPF related capabilities, but the cluster collector does not.
type capabilitiesExtension struct {
	logger *zap.Logger
}

var _ extension.Extension = (*capabilitiesExtension)(nil)

func (e *capabilitiesExtension) Start(_ context.Context, _ component.Host) error {
	if err := capabilities.AlignEffectiveCapToPermitted(); err != nil {
		e.logger.Warn("failed to align effective capabilities to permitted set", zap.Error(err))
	}
	return nil
}

func (e *capabilitiesExtension) Shutdown(_ context.Context) error {
	return nil
}
