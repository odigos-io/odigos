package testconnection

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/odigos-io/odigos/common/config/testconnection"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func freePort(t *testing.T) string {
	t.Helper()
	// let the OS assign the port number
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

func startNoOpOTLPGRPCReceiver(t *testing.T, ctx context.Context, endpoint string) {
	t.Helper()

	f := otlpreceiver.NewFactory()
	cfg := f.CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.Protocols.GRPC = configoptional.Some(configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  endpoint,
			Transport: confignet.TransportTypeTCP,
		},
	})
	cfg.Protocols.HTTP = configoptional.None[otlpreceiver.HTTPConfig]()

	sink := new(consumertest.TracesSink)
	r, err := f.CreateTraces(ctx, receivertest.NewNopSettings(f.Type()), cfg, sink)
	require.NoError(t, err)

	require.NoError(t, r.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, r.Shutdown(ctx)) })
}

func startNoOpOTLPHTTPReceiver(t *testing.T, ctx context.Context, endpoint string) {
	t.Helper()

	f := otlpreceiver.NewFactory()
	cfg := f.CreateDefaultConfig().(*otlpreceiver.Config)
	cfg.Protocols.GRPC = configoptional.None[configgrpc.ServerConfig]()

	httpCfg := cfg.Protocols.HTTP.GetOrInsertDefault()
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

func TestOTLPConnectionSuccess(t *testing.T) {
	ctx := context.Background()
	endpoint := freePort(t)
	startNoOpOTLPGRPCReceiver(t, ctx, endpoint)

	rawConfig := map[string]any{
		"endpoint": endpoint,
		"tls":      map[string]any{"insecure": true},
	}

	result := NewOTLPTester().TestExport(ctx, rawConfig)
	require.True(t, result.Succeeded, "message: %s", result.Message)
}

func TestOTLPConnectionRefused(t *testing.T) {
	ctx := context.Background()
	// a free port with nothing listening on it
	endpoint := freePort(t)

	rawConfig := map[string]any{
		"endpoint": endpoint,
		"tls":      map[string]any{"insecure": true},
	}

	result := NewOTLPTester().TestExport(ctx, rawConfig)
	require.False(t, result.Succeeded, "expected connection refused when no server is running")
	require.Equal(t, testconnection.FailedToConnect, result.Reason)
}

func TestOTLPHTTPConnectionSuccess(t *testing.T) {
	ctx := context.Background()
	endpoint := freePort(t)
	startNoOpOTLPHTTPReceiver(t, ctx, endpoint)

	rawConfig := map[string]any{
		"endpoint": fmt.Sprintf("http://%s", endpoint),
	}

	result := NewOTLPHTTPTester().TestExport(ctx, rawConfig)
	require.True(t, result.Succeeded, "message: %s", result.Message)
}

func TestOTLPHTTPConnectionRefused(t *testing.T) {
	ctx := context.Background()
	endpoint := freePort(t)

	rawConfig := map[string]any{
		"endpoint": fmt.Sprintf("http://%s", endpoint),
	}

	result := NewOTLPHTTPTester().TestExport(ctx, rawConfig)
	require.False(t, result.Succeeded, "expected connection refused when no server is running")
	require.Equal(t, testconnection.FailedToConnect, result.Reason)
}

func TestModifyConfigWrongType(t *testing.T) {
	require.Nil(t, modifyOTLPConfig(&dummyConfig{}), "otlp modify should return nil for wrong type")
	require.Nil(t, modifyOTLPHTTPConfig(&dummyConfig{}), "otlphttp modify should return nil for wrong type")
}

type dummyConfig struct{}
