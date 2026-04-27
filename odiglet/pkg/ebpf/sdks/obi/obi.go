package obi

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentation"

	"go.opentelemetry.io/obi/pkg/appolly/discover"
	"go.opentelemetry.io/obi/pkg/config"
	"go.opentelemetry.io/obi/pkg/instrumenter"
	"go.opentelemetry.io/obi/pkg/obi"
)

// OBIInstrumentationFactory creates instrumentations that add/remove PIDs on a shared
// DynamicPIDSelector while a single OBI instrumenter runs in the background.
// Requires OBI with DynamicPIDSelector support (e.g. go.opentelemetry.io/obi from main after PR 1388).
type OBIInstrumentationFactory struct {
	logger       *commonlogger.OdigosLogger
	obiCtx       context.Context
	obiCtxCancel context.CancelFunc

	selector *discover.DynamicPIDSelector
	obiCfg   *obi.Config
}

// NewOBIInstrumentationFactory returns a factory that uses the OBI SDK with a dynamic PID selector.
// The OBI instrumenter starts on first CreateInstrumentation; PIDs are added/removed via the selector.
func NewOBIInstrumentationFactory() *OBIInstrumentationFactory {
	return &OBIInstrumentationFactory{
		selector: discover.NewDynamicPIDSelector(),
		obiCfg:   obiConfigForOdigos(),
		logger:   commonlogger.LoggerCompat().With("subsystem", "opentelemetry-ebpf-instrumentation"),
	}
}

// obiConfigForOdigos returns a minimal OBI config that enables App O11y and exports to the node collector.
// PIDs are supplied dynamically via the DynamicPIDSelector.
func obiConfigForOdigos() *obi.Config {
	cfg := obi.DefaultConfig
	cfg.EBPF.ContextPropagation = config.ContextPropagationHeaders
	// Export traces to the node collector (same node as odiglet). Use http scheme for insecure gRPC.
	// Protocol is inferred from port (4317 -> gRPC) by OBI.
	cfg.Traces.TracesEndpoint = fmt.Sprintf("http://localhost:%d", consts.OTLPPort)
	return &cfg
}

// CreateInstrumentation starts the OBI instrumenter if it is not already running
// and returns an obiInstrumentation that allows adding/removing this PID using the dynamic selector.
func (f *OBIInstrumentationFactory) CreateInstrumentation(ctx context.Context, pid int, settings instrumentation.Settings) (instrumentation.Instrumentation, error) {
	if f.obiCtx == nil {
		obiCtx, obiCtxCancel := context.WithCancel(ctx)
		f.obiCtx = obiCtx
		f.obiCtxCancel = obiCtxCancel

		go func() {
			err := instrumenter.Run(f.obiCtx, f.obiCfg, instrumenter.WithDynamicPIDSelector(f.selector))
			if err != nil && f.obiCtx.Err() == nil {
				f.logger.Error("OBI instrumenter exited with error", "err", err)
			}
		}()
	}
	return &obiInstrumentation{selector: f.selector, pid: pid, factory: f}, nil
}

// obiInstrumentation implements instrumentation.Instrumentation; it only holds the factory's
// selector so Close(ctx) can call RemovePIDs for the PID.
type obiInstrumentation struct {
	selector *discover.DynamicPIDSelector
	pid      int
	factory  *OBIInstrumentationFactory
}

func (o *obiInstrumentation) Load(context.Context) (instrumentation.Status, error) {
	o.selector.AddPIDs(uint32(o.pid))
	return instrumentation.Status{}, nil
}

func (o *obiInstrumentation) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (o *obiInstrumentation) Close(_ context.Context) error {
	o.selector.RemovePIDs(uint32(o.pid))
	if _, ok := o.selector.GetPIDs(); !ok {
		if o.factory.obiCtxCancel != nil {
			o.factory.obiCtxCancel()
			o.factory.obiCtxCancel = nil
			o.factory.obiCtx = nil
		}
	}
	return nil
}

func (o *obiInstrumentation) ApplyConfig(context.Context, instrumentation.Config) error {
	return nil
}
