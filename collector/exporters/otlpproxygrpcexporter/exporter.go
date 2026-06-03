package otlpproxygrpcexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/grpc"
)

// proxyExporter is an OTLP/gRPC exporter whose client connection is dialed
// through an HTTP CONNECT proxy when proxy_url is set.
type proxyExporter struct {
	cfg      *Config
	settings exporter.Settings

	clientConn   *grpc.ClientConn
	traceClient  ptraceotlp.GRPCClient
	metricClient pmetricotlp.GRPCClient
	logClient    plogotlp.GRPCClient
}

func newExporter(cfg *Config, set exporter.Settings) *proxyExporter {
	return &proxyExporter{cfg: cfg, settings: set}
}

// start builds the gRPC client connection, injecting the CONNECT-proxy dialer
// when proxy_url is set. Rebuilt on every collector config reload, so toggling
// or changing the proxy takes effect without a process restart.
func (e *proxyExporter) start(ctx context.Context, host component.Host) error {
	var opts []configgrpc.ToClientConnOption
	if e.cfg.ProxyURL != "" {
		opts = append(opts, configgrpc.WithGrpcDialOption(
			grpc.WithContextDialer(connectProxyDialer(e.cfg.ProxyURL)),
		))
	}
	conn, err := e.cfg.ClientConfig.ToClientConn(ctx, host.GetExtensions(), e.settings.TelemetrySettings, opts...)
	if err != nil {
		return err
	}
	e.clientConn = conn
	e.traceClient = ptraceotlp.NewGRPCClient(conn)
	e.metricClient = pmetricotlp.NewGRPCClient(conn)
	e.logClient = plogotlp.NewGRPCClient(conn)
	return nil
}

func (e *proxyExporter) shutdown(context.Context) error {
	if e.clientConn != nil {
		return e.clientConn.Close()
	}
	return nil
}

func (e *proxyExporter) pushTraces(ctx context.Context, td ptrace.Traces) error {
	_, err := e.traceClient.Export(ctx, ptraceotlp.NewExportRequestFromTraces(td))
	return err
}

func (e *proxyExporter) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	_, err := e.metricClient.Export(ctx, pmetricotlp.NewExportRequestFromMetrics(md))
	return err
}

func (e *proxyExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	_, err := e.logClient.Export(ctx, plogotlp.NewExportRequestFromLogs(ld))
	return err
}
