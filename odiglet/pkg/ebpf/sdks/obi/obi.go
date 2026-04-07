package obi

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentation"

	"go.opentelemetry.io/obi/pkg/appolly/discover"
	"go.opentelemetry.io/obi/pkg/config"
	"go.opentelemetry.io/obi/pkg/instrumenter"
	"go.opentelemetry.io/obi/pkg/obi"
)

// obiLogger must be called after commonlogger.Init (e.g. from CreateInstrumentation), not at package
// init: LoggerCompat() returns a nop logger until Init runs, and main imports this package before Init.
func obiLogger() *commonlogger.OdigosLogger {
	return commonlogger.LoggerCompat().With("subsystem", "obisdk")
}

// setObiSlogDefault installs the global slog logger OBI uses (e.g. instrumenter.Run's slog.Debug).
// The standalone obi CLI does this in cmd/obi/main.go; the library entrypoint does not, so odiglet must.
// Level follows OTEL_LOG_LEVEL (e.g. debug, info). Output is always JSON to match the rest of odiglet.
func setObiSlogDefault() {
	var lv slog.LevelVar
	lv.Set(slog.LevelInfo)
	if s := strings.TrimSpace(os.Getenv("OTEL_LOG_LEVEL")); s != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(strings.ToUpper(s))); err == nil {
			lv.Set(parsed)
		}
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: &lv})
	slog.SetDefault(slog.New(h).With("subsystem", "obisdk", "logsource", "upstream"))
}

// OBIInstrumentationFactory creates instrumentations that add/remove PIDs on a shared
// DynamicPIDSelector while a single OBI instrumenter runs in the background.
// Requires OBI with DynamicPIDSelector support (e.g. go.opentelemetry.io/obi from main after PR 1388).
type OBIInstrumentationFactory struct {
	ctx          context.Context
	obiCtx       context.Context
	obiCtxCancel context.CancelFunc

	selector *discover.DynamicPIDSelector
	obiCfg   *obi.Config
}

// NewOBIInstrumentationFactory returns a factory that uses the OBI SDK with a dynamic PID selector.
// The OBI instrumenter starts on first CreateInstrumentation; PIDs are added/removed via the selector.
func NewOBIInstrumentationFactory(ctx context.Context) *OBIInstrumentationFactory {
	return &OBIInstrumentationFactory{
		ctx:      ctx,
		selector: discover.NewDynamicPIDSelector(),
		obiCfg:   obiConfigForOdigos(),
	}
}

// obiConfigForOdigos returns a minimal OBI config that enables App O11y and exports to the node collector.
// PIDs are supplied dynamically via the DynamicPIDSelector.
// OBI log/slog uses OTEL_LOG_LEVEL; see setObiSlogDefault.
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
	obiLogger().Info("Creating OBI instrumentation", "pid", pid, "serviceName", settings.ServiceName)
	if f.obiCtx == nil {
		obiCtx, obiCtxCancel := context.WithCancel(f.ctx)
		f.obiCtx = obiCtx
		f.obiCtxCancel = obiCtxCancel

		go func() {
			setObiSlogDefault()
			obiLogger().Info("Starting OBI instrumenter", "pid", pid, "serviceName", settings.ServiceName)
			err := instrumenter.Run(f.obiCtx, f.obiCfg, instrumenter.WithDynamicPIDSelector(f.selector))
			if err != nil && f.obiCtx.Err() == nil {
				obiLogger().Error("OBI instrumenter exited with error", "err", err)
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
	obiLogger().Info("Removed PID", "pid", o.pid)
	if _, ok := o.selector.GetPIDs(); !ok {
		obiLogger().Info("No PIDs left, stopping OBI instrumenter")
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
