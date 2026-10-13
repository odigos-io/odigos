package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
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

func TestLegacySplunkConfigLoads(t *testing.T) {
	dest := &testconnection.TestConnectionConfig{
		DestinationType: string(common.SplunkDestinationType), ID: "legacy-splunk",
		Config:  map[string]string{"SPLUNK_REALM": "us0"},
		Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}
	t.Setenv(destinationconfig.SecretEnvVarName("SPLUNK_ACCESS_TOKEN", dest.GetID()), "test-token")
	cfg := destinationconfig.Config{
		Receivers: destinationconfig.GenericMap{"otlp": destinationconfig.GenericMap{"protocols": destinationconfig.GenericMap{"grpc": destinationconfig.GenericMap{"endpoint": "127.0.0.1:0"}}}},
		Exporters: destinationconfig.GenericMap{},
		Service:   destinationconfig.Service{Pipelines: map[string]destinationconfig.Pipeline{}},
	}
	pipelines, err := (&destinationconfig.Splunk{}).ModifyConfig(dest, &cfg)
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	pipeline := cfg.Service.Pipelines[pipelines[0]]
	pipeline.Receivers = []string{"otlp"}
	cfg.Service.Pipelines[pipelines[0]] = pipeline
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "splunk.yaml")
	require.NoError(t, os.WriteFile(path, data, 0600))
	col, err := otelcol.NewCollector(otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(), Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{ResolverSettings: confmap.ResolverSettings{
			URIs: []string{path}, DefaultScheme: "env",
			ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
		}},
	})
	require.NoError(t, err)
	require.NoError(t, col.DryRun(context.Background()))
}

// spDryRunDestination renders dest through configurer and asks the real component
// registry to load the result, which is how CORE-1496 ("unknown type: sapm") would have
// been caught before it reached a gateway.
func spDryRunDestination(t *testing.T, configurer destinationconfig.Configer, dest *testconnection.TestConnectionConfig) error {
	t.Helper()

	t.Setenv(destinationconfig.SecretEnvVarName("SPLUNK_ACCESS_TOKEN", dest.GetID()), "test-token")
	cfg := destinationconfig.Config{
		Receivers: destinationconfig.GenericMap{"otlp": destinationconfig.GenericMap{"protocols": destinationconfig.GenericMap{"grpc": destinationconfig.GenericMap{"endpoint": "127.0.0.1:0"}}}},
		Exporters: destinationconfig.GenericMap{},
		Service:   destinationconfig.Service{Pipelines: map[string]destinationconfig.Pipeline{}},
	}
	pipelines, err := configurer.ModifyConfig(dest, &cfg)
	require.NoError(t, err)
	require.Len(t, pipelines, 1)

	pipeline := cfg.Service.Pipelines[pipelines[0]]
	pipeline.Receivers = []string{"otlp"}
	cfg.Service.Pipelines[pipelines[0]] = pipeline

	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "splunk.yaml")
	require.NoError(t, os.WriteFile(path, data, 0600))

	col, err := otelcol.NewCollector(otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(), Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{ResolverSettings: confmap.ResolverSettings{
			URIs: []string{path}, DefaultScheme: "env",
			ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
		}},
	})
	require.NoError(t, err)
	return col.DryRun(context.Background())
}

// spSelfSignedCAPem is a PEM-encoded self-signed certificate, so the SPLUNK_OTLP_CA_PEM
// fixture carries something configtls could actually load rather than a placeholder.
func spSelfSignedCAPem(t *testing.T) string {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "odigos-splunk-smoketest-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// TestSplunkConfigLoadsWithEveryCatalogOption loads a config for every optional field the
// splunkotlp catalog entry can emit. #6021 routed the legacy splunk destination through
// SplunkOTLP.modifyConfig, so both destination types now reach these fields and a
// collector upgrade that stops accepting one of them breaks both.
func TestSplunkConfigLoadsWithEveryCatalogOption(t *testing.T) {
	configurers := map[string]struct {
		configurer destinationconfig.Configer
		destType   common.DestinationType
	}{
		"legacy splunk": {&destinationconfig.Splunk{}, common.SplunkDestinationType},
		"splunkotlp":    {&destinationconfig.SplunkOTLP{}, common.SplunkOTLPDestinationType},
	}

	options := map[string]map[string]string{
		"no optional fields": {},
		// "none" is the catalog's initialValue, so it is what a destination created from
		// the UI with default settings carries.
		"compression none": {"SPLUNK_OTLP_COMPRESSION": "none"},
		"compression gzip": {"SPLUNK_OTLP_COMPRESSION": "gzip"},
		"tls":              {"SPLUNK_OTLP_TLS_ENABLED": "true"},
		"tls with skip verify": {
			"SPLUNK_OTLP_TLS_ENABLED":          "true",
			"SPLUNK_OTLP_INSECURE_SKIP_VERIFY": "true",
		},
	}

	for configurerName, c := range configurers {
		for optionName, option := range options {
			t.Run(configurerName+"/"+optionName, func(t *testing.T) {
				destConfig := map[string]string{"SPLUNK_REALM": "us0"}
				for k, v := range option {
					destConfig[k] = v
				}
				dest := &testconnection.TestConnectionConfig{
					DestinationType: string(c.destType), ID: "splunk-smoketest",
					Config:  destConfig,
					Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
				}
				require.NoError(t, spDryRunDestination(t, c.configurer, dest))
			})
		}

		t.Run(configurerName+"/tls with ca pem", func(t *testing.T) {
			dest := &testconnection.TestConnectionConfig{
				DestinationType: string(c.destType), ID: "splunk-smoketest",
				Config: map[string]string{
					"SPLUNK_REALM":            "us0",
					"SPLUNK_OTLP_TLS_ENABLED": "true",
					"SPLUNK_OTLP_CA_PEM":      spSelfSignedCAPem(t),
				},
				Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			}
			require.NoError(t, spDryRunDestination(t, c.configurer, dest))
		})
	}
}

// TestSplunkConfigLoadRejectsAnUnsupportedCompression is the control for the test above:
// it proves DryRun actually validates the exporter's fields, and that the configured
// compression reaches the exporter rather than being dropped on the way.
func TestSplunkConfigLoadRejectsAnUnsupportedCompression(t *testing.T) {
	dest := &testconnection.TestConnectionConfig{
		DestinationType: string(common.SplunkOTLPDestinationType), ID: "splunk-bad-compression",
		Config:  map[string]string{"SPLUNK_REALM": "us0", "SPLUNK_OTLP_COMPRESSION": "not-a-compression"},
		Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
	}
	err := spDryRunDestination(t, &destinationconfig.SplunkOTLP{}, dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not-a-compression")
}
