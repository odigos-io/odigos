package testconnection

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// startNoOpOTLPHTTPReceiver spins up an OTLP HTTP receiver backed by a no-op
// consumer and returns the address it is listening on.
func startNoOpOTLPHTTPReceiver(t *testing.T, ctx context.Context, endpoint string) {
	t.Helper()

	f := otlpreceiver.NewFactory()
	cfg := f.CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.GRPC = configoptional.None[configgrpc.ServerConfig]()

	httpCfg := cfg.HTTP.GetOrInsertDefault()
	httpCfg.ServerConfig.NetAddr = confignet.AddrConfig{
		Endpoint:  endpoint,
		Transport: confignet.TransportTypeTCP,
	}

	sink := new(consumertest.TracesSink)
	r, err := f.CreateTraces(ctx, receivertest.NewNopSettings(f.Type()), cfg, sink)
	require.NoError(t, err)

	require.NoError(t, r.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, r.Shutdown(ctx)) })
}

func TestOTLPHTTPConnectionSuccess(t *testing.T) {
	ctx := context.Background()
	endpoint := freePort(t)

	startNoOpOTLPHTTPReceiver(t, ctx, endpoint)

	tester := NewOTLPHTTPTester()

	factory := tester.Factory()
	require.NotNil(t, factory)

	defaultCfg := factory.CreateDefaultConfig()
	modifiedCfg := tester.ModifyConfigForConnectionTest(defaultCfg)
	require.NotNil(t, modifiedCfg)

	httpCfg, ok := modifiedCfg.(*otlphttpexporter.Config)
	require.True(t, ok)

	httpCfg.ClientConfig.Endpoint = fmt.Sprintf("http://%s", endpoint)

	exp, err := factory.CreateTraces(ctx, exportertest.NewNopSettings(factory.Type()), httpCfg)
	require.NoError(t, err)

	require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(ctx)) })

	require.NoError(t, exp.ConsumeTraces(ctx, ptrace.NewTraces()))
}

func TestOTLPHTTPConnectionRefused(t *testing.T) {
	ctx := context.Background()
	endpoint := freePort(t)

	tester := NewOTLPHTTPTester()

	factory := tester.Factory()
	defaultCfg := factory.CreateDefaultConfig()
	modifiedCfg := tester.ModifyConfigForConnectionTest(defaultCfg)

	httpCfg := modifiedCfg.(*otlphttpexporter.Config)
	httpCfg.ClientConfig.Endpoint = fmt.Sprintf("http://%s", endpoint)

	exp, err := factory.CreateTraces(ctx, exportertest.NewNopSettings(factory.Type()), httpCfg)
	require.NoError(t, err)

	require.NoError(t, exp.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, exp.Shutdown(ctx)) })

	err = exp.ConsumeTraces(ctx, ptrace.NewTraces())
	require.Error(t, err, "expected connection refused when no server is running")
}

func TestHTTPModifyConfigForConnectionTest_WrongType(t *testing.T) {
	tester := NewOTLPHTTPTester()
	result := tester.ModifyConfigForConnectionTest(&dummyHTTPConfig{})
	require.Nil(t, result, "should return nil for non-otlphttp config")
}

type dummyHTTPConfig struct{}
