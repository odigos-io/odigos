package testconnection

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func freePort(t *testing.T) string {
	t.Helper()
	// let the port number to be assigned by the OS
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// startNoOpOTLPReceiver spins up an OTLP gRPC receiver backed by a no-op
// consumer and returns the address it is listening on plus a teardown func.
func startNoOpOTLPReceiver(t *testing.T, ctx context.Context, endpoint string) {
	t.Helper()

	f := otlpreceiver.NewFactory()
	cfg := f.CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.GRPC = configoptional.Some(configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  endpoint,
			Transport: confignet.TransportTypeTCP,
		},
	})
	cfg.HTTP = configoptional.None[otlpreceiver.HTTPConfig]()

	sink := new(consumertest.TracesSink)
	r, err := f.CreateTraces(ctx, receivertest.NewNopSettings(f.Type()), cfg, sink)
	require.NoError(t, err)

	require.NoError(t, r.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, r.Shutdown(ctx)) })
}

func TestOTLPConnectionSuccess(t *testing.T) {
	ctx := context.Background()
	endpoint := freePort(t)

	startNoOpOTLPReceiver(t, ctx, endpoint)

	tester := NewOTLPTester()

	factory := tester.Factory()
	require.NotNil(t, factory)

	defaultCfg := factory.CreateDefaultConfig()
	modifiedCfg := tester.ModifyConfigForConnectionTest(defaultCfg)
	require.NotNil(t, modifiedCfg)

	otlpCfg, ok := modifiedCfg.(*otlpexporter.Config)
	require.True(t, ok)

	otlpCfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: endpoint,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}

	exp, err := factory.CreateTraces(ctx, exportertest.NewNopSettings(factory.Type()), otlpCfg)
	require.NoError(t, err)

	require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(ctx)) })

	require.NoError(t, exp.ConsumeTraces(ctx, ptrace.NewTraces()))
}

func TestOTLPConnectionRefused(t *testing.T) {
	ctx := context.Background()
	// Use a port with nothing listening on it.
	endpoint := freePort(t)

	tester := NewOTLPTester()

	factory := tester.Factory()
	defaultCfg := factory.CreateDefaultConfig()
	modifiedCfg := tester.ModifyConfigForConnectionTest(defaultCfg)

	otlpCfg := modifiedCfg.(*otlpexporter.Config)
	otlpCfg.ClientConfig = configgrpc.ClientConfig{
		Endpoint: endpoint,
		TLS: configtls.ClientConfig{
			Insecure: true,
		},
	}

	exp, err := factory.CreateTraces(ctx, exportertest.NewNopSettings(factory.Type()), otlpCfg)
	require.NoError(t, err)

	require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(ctx)) })

	err = exp.ConsumeTraces(ctx, ptrace.NewTraces())
	require.Error(t, err, "expected connection refused when no server is running")
}
