package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	envprovider "go.opentelemetry.io/collector/confmap/provider/envprovider"
	fileprovider "go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/odigos-io/odigos/common"
	destinationconfig "github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/config/testconnection"
)

// TestFullSignalConfigStartsSuccessfully builds a real Collector — using the actual
// component registry from components(), not fakes — from a config that puts the
// odigostrafficmetrics processor in a pipeline for every signal, including profiles.
//
// This reproduces, at test time, the exact failure mode that crash-looped odigos-gateway
// in production: a processor added to a pipeline for a signal its factory doesn't support
// only surfaces as "telemetry type is not supported" deep inside service.New, which neither
// per-processor unit tests nor otelcoltest.LoadConfigAndValidate (schema validation only,
// against Nop factories in existing usages) can catch.
func TestFullSignalConfigStartsSuccessfully(t *testing.T) {
	set := otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{"testdata/full-signal-config.yaml"},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
					envprovider.NewFactory(),
				},
				DefaultScheme: "env",
			},
		},
	}

	col, err := otelcol.NewCollector(set)
	require.NoError(t, err)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- col.Run(context.Background())
	}()

	require.Eventually(t, func() bool {
		state := col.GetState()
		return state == otelcol.StateRunning || state == otelcol.StateClosed
	}, 10*time.Second, 50*time.Millisecond, "collector did not reach a running or closed state")

	if col.GetState() == otelcol.StateRunning {
		col.Shutdown()
	}

	select {
	case err := <-runErrCh:
		assert.NoError(t, err, "collector failed to build pipelines for a config where every "+
			"signal (traces/metrics/logs/profiles) routes through odigostrafficmetrics — "+
			"this is the exact error class that crash-looped odigos-gateway in production")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for collector shutdown")
	}
}

// A SignalFx destination that selected traces must render a traces pipeline the gateway can
// actually build, and it must not be the signalfx exporter: that exporter's traces capability is
// deprecated for removal in December 2026, and it already discards every span it is handed
// (noOpProcessTraces), so it would reproduce the silent drop rather than fix it.
func TestSignalFxTracesConfigLoads(t *testing.T) {
	dest := &testconnection.TestConnectionConfig{
		DestinationType: string(common.SignalFxDestinationType), ID: "signalfx-traces",
		Config:  map[string]string{"SIGNALFX_REALM": "us0"},
		Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}
	t.Setenv(destinationconfig.SecretEnvVarName("SIGNALFX_ACCESS_TOKEN", dest.GetID()), "test-token")
	cfg := newSmoketestConfig()
	pipelines, err := (&destinationconfig.SignalFx{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"traces/signalfx-signalfx-traces"}, pipelines)
	assert.Equal(t, []string{"otlp_http/signalfx-signalfx-traces"}, cfg.Service.Pipelines[pipelines[0]].Exporters)

	attachOtlpReceiver(&cfg, pipelines[0])
	assert.NoError(t, runCollectorConfig(t, cfg))
}

func newSmoketestConfig() destinationconfig.Config {
	return destinationconfig.Config{
		Receivers: destinationconfig.GenericMap{"otlp": destinationconfig.GenericMap{"protocols": destinationconfig.GenericMap{"grpc": destinationconfig.GenericMap{"endpoint": "127.0.0.1:0"}}}},
		Exporters: destinationconfig.GenericMap{},
		Service:   destinationconfig.Service{Pipelines: map[string]destinationconfig.Pipeline{}},
	}
}

func attachOtlpReceiver(cfg *destinationconfig.Config, pipelineName string) {
	pipeline := cfg.Service.Pipelines[pipelineName]
	pipeline.Receivers = []string{"otlp"}
	cfg.Service.Pipelines[pipelineName] = pipeline
}

func newSmoketestCollector(t *testing.T, cfg destinationconfig.Config) *otelcol.Collector {
	t.Helper()
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, data, 0600))
	col, err := otelcol.NewCollector(otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(), Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{ResolverSettings: confmap.ResolverSettings{
			URIs: []string{path}, DefaultScheme: "env",
			ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
		}},
	})
	require.NoError(t, err)
	return col
}

func dryRunCollectorConfig(t *testing.T, cfg destinationconfig.Config) error {
	t.Helper()
	return newSmoketestCollector(t, cfg).DryRun(context.Background())
}

// runCollectorConfig starts a real Collector and returns the error it exits with, so a pipeline
// that cannot be built — which DryRun does not catch, since it only validates the config schema —
// surfaces as a test failure.
func runCollectorConfig(t *testing.T, cfg destinationconfig.Config) error {
	t.Helper()
	col := newSmoketestCollector(t, cfg)

	runErrCh := make(chan error, 1)
	go func() { runErrCh <- col.Run(context.Background()) }()

	require.Eventually(t, func() bool {
		state := col.GetState()
		return state == otelcol.StateRunning || state == otelcol.StateClosed
	}, 10*time.Second, 50*time.Millisecond, "collector did not reach a running or closed state")

	if col.GetState() == otelcol.StateRunning {
		col.Shutdown()
	}

	select {
	case err := <-runErrCh:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for collector shutdown")
		return nil
	}
}

func TestLegacySplunkConfigLoads(t *testing.T) {
	dest := &testconnection.TestConnectionConfig{
		DestinationType: string(common.SplunkDestinationType), ID: "legacy-splunk",
		Config:  map[string]string{"SPLUNK_REALM": "us0"},
		Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}
	t.Setenv(destinationconfig.SecretEnvVarName("SPLUNK_ACCESS_TOKEN", dest.GetID()), "test-token")
	cfg := newSmoketestConfig()
	pipelines, err := (&destinationconfig.Splunk{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	attachOtlpReceiver(&cfg, pipelines[0])
	require.NoError(t, dryRunCollectorConfig(t, cfg))
}
