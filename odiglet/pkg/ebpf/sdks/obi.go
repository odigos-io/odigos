package sdks

import (
	"context"
	"fmt"
	"sync"

	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentation"

	"go.opentelemetry.io/obi/pkg/appolly/discover"
	"go.opentelemetry.io/obi/pkg/config"
	"go.opentelemetry.io/obi/pkg/instrumenter"
	"go.opentelemetry.io/obi/pkg/obi"
)

var obiLog = commonlogger.LoggerCompat().With("subsystem", "ebpfobi")

// OBIInstrumentationFactory creates instrumentations that add/remove PIDs on a shared
// DynamicPIDSelector while a single OBI instrumenter runs in the background.
// Requires OBI with DynamicPIDSelector support (e.g. go.opentelemetry.io/obi from main after PR 1388).
type OBIInstrumentationFactory struct {
	once     sync.Once
	selector *discover.DynamicPIDSelector
	obiCfg   *obi.Config
	startErr error
}

// NewOBIInstrumentationFactory returns a factory that uses the OBI SDK with a dynamic PID selector.
// The OBI instrumenter starts on first CreateInstrumentation; PIDs are added/removed via the selector.
func NewOBIInstrumentationFactory() *OBIInstrumentationFactory {
	return &OBIInstrumentationFactory{}
}

// ensureOBIStarted starts the OBI instrumenter once with the dynamic PID selector.
// It must be called with a context that is canceled on process shutdown.
func (f *OBIInstrumentationFactory) ensureOBIStarted(ctx context.Context) error {
	f.once.Do(func() {
		f.selector = discover.NewDynamicPIDSelector()
		f.obiCfg = obiConfigForOdigos()
		go func() {
			err := instrumenter.Run(ctx, f.obiCfg, instrumenter.WithDynamicPIDSelector(f.selector))
			if err != nil && ctx.Err() == nil {
				obiLog.Error("OBI instrumenter exited with error", "err", err)
			}
		}()
	})
	return f.startErr
}

// obiConfigForOdigos returns a minimal OBI config that enables App O11y and exports to the node collector.
// PIDs are supplied dynamically via the DynamicPIDSelector (no default GlobDefinitionCriteria needed on this branch).
func obiConfigForOdigos() *obi.Config {
	cfg := obi.DefaultConfig
	cfg.EBPF.ContextPropagation = config.ContextPropagationHeaders
	// Export traces to the node collector (same node as odiglet). Use http scheme for insecure gRPC.
	// Protocol is inferred from port (4317 -> gRPC) by OBI.
	cfg.Traces.TracesEndpoint = fmt.Sprintf("http://localhost:%d", consts.OTLPPort)
	return &cfg
}

// CreateInstrumentation adds the process PID to the OBI dynamic selector and returns an
// instrumentation handle whose Close(ctx, pid) removes that PID from the same selector.
func (f *OBIInstrumentationFactory) CreateInstrumentation(ctx context.Context, pid int, settings instrumentation.Settings) (instrumentation.Instrumentation, error) {
	if err := f.ensureOBIStarted(ctx); err != nil {
		return nil, err
	}
	f.selector.AddPIDs(uint32(pid))
	return &obiInstrumentation{selector: f.selector}, nil
}

// obiInstrumentation implements instrumentation.Instrumentation; it only holds the factory's
// selector so Close(ctx, pid) can call RemovePIDs for the manager-provided PID.
type obiInstrumentation struct {
	selector *discover.DynamicPIDSelector
}

func (o *obiInstrumentation) Load(context.Context) (instrumentation.Status, error) {
	return instrumentation.Status{}, nil
}

func (o *obiInstrumentation) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (o *obiInstrumentation) Close(_ context.Context, pid int) error {
	o.selector.RemovePIDs(uint32(pid))
	return nil
}

func (o *obiInstrumentation) ApplyConfig(context.Context, instrumentation.Config) error {
	return nil
}
