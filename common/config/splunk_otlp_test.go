package config

import (
	"maps"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	spDestID = "odigos.io.dest.splunk-abc123"
	// Spelled out rather than built with SecretEnvPlaceholder so that renaming the
	// field, the prefix or the ID-sanitising rule is visible here: the gateway resolves
	// this placeholder against an env var the k8s layer mounts under that exact name.
	spTokenPlaceholder = "${ODIGOS_DEST_odigos_io_dest_splunk_abc123_SPLUNK_ACCESS_TOKEN}"
	spTracesEndpoint   = "https://ingest.eu0.signalfx.com/v2/trace/otlp"
)

// spConfigurers is every configurer backed by the shared SplunkOTLP.modifyConfig.
// The pipeline prefix each one passes is the only thing that distinguishes them, so
// every table below runs against both.
func spConfigurers() map[string]Configer {
	return map[string]Configer{
		"legacy splunk": &Splunk{},
		"splunkotlp":    &SplunkOTLP{},
	}
}

func spEmptyConfig() Config {
	return Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
}

func spDest(cfg map[string]string) *mockDestination {
	return &mockDestination{id: spDestID, config: cfg, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}}
}

// TestSplunkPipelineNameIsPrefixedWithTheDestinationType pins the one value that
// differs between the two configurers. #6021 made the legacy splunk destination reuse
// SplunkOTLP.modifyConfig and pass its own prefix, so a swapped or copy-pasted literal
// would silently rename the pipeline of every existing splunk Destination CR.
func TestSplunkPipelineNameIsPrefixedWithTheDestinationType(t *testing.T) {
	// Literal expectations: both halves of the derivation below come from production
	// code, so they would agree with each other even if both were renamed together.
	wantLiteral := map[string]string{
		"legacy splunk": "traces/splunk-" + spDestID,
		"splunkotlp":    "traces/splunkotlp-" + spDestID,
	}

	// Collected from the values production code actually returned, so that a prefix
	// shared by both configurers collapses this set.
	got := map[string]bool{}
	for name, configurer := range spConfigurers() {
		t.Run(name, func(t *testing.T) {
			cfg := spEmptyConfig()
			pipelines, err := configurer.ModifyConfig(spDest(map[string]string{splunkRealm: "eu0"}), &cfg)
			require.NoError(t, err)
			require.Len(t, pipelines, 1)
			got[pipelines[0]] = true

			assert.Equal(t, wantLiteral[name], pipelines[0])
			// The prefix is not a free-form label: it is the destination type, which is
			// what makes two Splunk destinations of different types land in different
			// pipelines.
			assert.Equal(t, "traces/"+string(configurer.DestType())+"-"+spDestID, pipelines[0])
			assert.Contains(t, cfg.Service.Pipelines, pipelines[0])
		})
	}
	assert.Len(t, got, len(spConfigurers()), "the two configurers must not produce the same pipeline name")
}

// TestSplunkLegacyAndOTLPProduceIdenticalExporters is the contract #6021 created: the
// legacy destination is now nothing but SplunkOTLP with a different pipeline prefix.
// Any option handled for one and not the other is a divergence this catches.
func TestSplunkLegacyAndOTLPProduceIdenticalExporters(t *testing.T) {
	fixtures := map[string]map[string]string{
		"realm only":      {splunkRealm: "eu0"},
		"compression":     {splunkRealm: "eu0", splunkOtlpCompression: "gzip"},
		"tls":             {splunkRealm: "eu0", splunkOtlpTlsKey: "true"},
		"tls with ca pem": {splunkRealm: "eu0", splunkOtlpTlsKey: "true", splunkOtlpCaPemKey: "-----BEGIN CERTIFICATE-----"},
		"tls skip verify": {splunkRealm: "eu0", splunkOtlpTlsKey: "true", splunkOtlpInsecureSkipVerify: "true"},
	}

	// Anti-vacuity: two configurers that both emit nothing also "agree". Require the
	// fixtures to actually exercise the optional blocks.
	sawTLS, sawCompression := false, false

	for name, destConfig := range fixtures {
		t.Run(name, func(t *testing.T) {
			legacyCfg, otlpCfg := spEmptyConfig(), spEmptyConfig()

			legacyPipelines, err := (&Splunk{}).ModifyConfig(spDest(destConfig), &legacyCfg)
			require.NoError(t, err)
			otlpPipelines, err := (&SplunkOTLP{}).ModifyConfig(spDest(destConfig), &otlpCfg)
			require.NoError(t, err)

			assert.Equal(t, otlpCfg.Exporters, legacyCfg.Exporters)
			require.Len(t, legacyPipelines, 1)
			require.Len(t, otlpPipelines, 1)
			assert.NotEqual(t, otlpPipelines[0], legacyPipelines[0])
			// Differing only in name, the two pipelines must still export through the
			// same exporter definition.
			assert.Equal(t,
				otlpCfg.Service.Pipelines[otlpPipelines[0]].Exporters,
				legacyCfg.Service.Pipelines[legacyPipelines[0]].Exporters)

			exporter := legacyCfg.Exporters["otlp_http/"+spDestID].(GenericMap)
			if _, ok := exporter["tls"]; ok {
				sawTLS = true
			}
			if _, ok := exporter["compression"]; ok {
				sawCompression = true
			}
		})
	}

	assert.True(t, sawTLS, "no fixture produced a tls block, so the agreement above is vacuous for TLS")
	assert.True(t, sawCompression, "no fixture produced a compression key")
}

// TestSplunkOTLPExporterOptions asserts the whole exporter map per fixture, so a value
// written under the wrong key — or carried over from a sibling option — is visible.
// None of these options had any coverage: SplunkOTLP.ModifyConfig was at 0%, and #6021
// routed the legacy destination through the same code.
func TestSplunkOTLPExporterOptions(t *testing.T) {
	base := func() GenericMap {
		return GenericMap{
			"headers":         GenericMap{"X-SF-Token": spTokenPlaceholder},
			"traces_endpoint": spTracesEndpoint,
		}
	}
	withKeys := func(extra GenericMap) GenericMap {
		out := base()
		maps.Copy(out, extra)
		return out
	}

	tests := []struct {
		name   string
		config map[string]string
		want   GenericMap
	}{
		{
			name:   "no optional fields",
			config: map[string]string{},
			want:   base(),
		},
		{
			name:   "compression gzip",
			config: map[string]string{splunkOtlpCompression: "gzip"},
			want:   withKeys(GenericMap{"compression": "gzip"}),
		},
		{
			// The catalog entry for splunkotlp sets initialValue "none", so this is what
			// the UI sends for a destination created with the defaults.
			name:   "compression none",
			config: map[string]string{splunkOtlpCompression: "none"},
			want:   withKeys(GenericMap{"compression": "none"}),
		},
		{
			// Presence, not emptiness, gates compression — unlike ca_pem below.
			name:   "compression present but empty",
			config: map[string]string{splunkOtlpCompression: ""},
			want:   withKeys(GenericMap{"compression": ""}),
		},
		{
			name:   "tls enabled",
			config: map[string]string{splunkOtlpTlsKey: "true"},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false}}),
		},
		{
			name:   "tls enabled with ca pem",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpCaPemKey: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"},
			want: withKeys(GenericMap{"tls": GenericMap{
				"insecure": false,
				"ca_pem":   "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
			}}),
		},
		{
			name:   "tls enabled with empty ca pem",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpCaPemKey: ""},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false}}),
		},
		{
			name:   "tls enabled with skip verify true",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpInsecureSkipVerify: "true"},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false, "insecure_skip_verify": true}}),
		},
		{
			// An explicit "false" is written through as false rather than omitted, which
			// is what distinguishes it from the empty case below.
			name:   "tls enabled with skip verify false",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpInsecureSkipVerify: "false"},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false, "insecure_skip_verify": false}}),
		},
		{
			name:   "tls enabled with empty skip verify",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpInsecureSkipVerify: ""},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false}}),
		},
		{
			// Destination.spec.data has no validating webhook, so a non-boolean can reach
			// here; parseBool swallows the error and the secure value wins.
			name:   "tls enabled with unparsable skip verify",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpInsecureSkipVerify: "yes"},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false, "insecure_skip_verify": false}}),
		},
		{
			// Every spelling strconv.ParseBool accepts disables verification, which is
			// what separates this field from the literal "true" the TLS gate requires.
			name:   "tls enabled with skip verify spelled as 1",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpInsecureSkipVerify: "1"},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false, "insecure_skip_verify": true}}),
		},
		{
			name:   "tls enabled with skip verify spelled as TRUE",
			config: map[string]string{splunkOtlpTlsKey: "true", splunkOtlpInsecureSkipVerify: "TRUE"},
			want:   withKeys(GenericMap{"tls": GenericMap{"insecure": false, "insecure_skip_verify": true}}),
		},
		{
			name: "every option at once",
			config: map[string]string{
				splunkOtlpCompression:        "gzip",
				splunkOtlpTlsKey:             "true",
				splunkOtlpCaPemKey:           "CA",
				splunkOtlpInsecureSkipVerify: "true",
			},
			want: withKeys(GenericMap{
				"compression": "gzip",
				"tls":         GenericMap{"insecure": false, "ca_pem": "CA", "insecure_skip_verify": true},
			}),
		},
		{
			// The CA certificate and the skip-verify choice are dropped unless TLS is
			// enabled. The catalog hides both fields behind a renderCondition on
			// SPLUNK_OTLP_TLS_ENABLED, but a Destination CR applied directly can set them.
			name:   "ca pem and skip verify are dropped when tls is disabled",
			config: map[string]string{splunkOtlpTlsKey: "false", splunkOtlpCaPemKey: "CA", splunkOtlpInsecureSkipVerify: "true"},
			want:   base(),
		},
		{
			name:   "ca pem and skip verify are dropped when tls is unset",
			config: map[string]string{splunkOtlpCaPemKey: "CA", splunkOtlpInsecureSkipVerify: "true"},
			want:   base(),
		},
		{
			// The TLS gate is a literal "true" comparison while insecure_skip_verify goes
			// through parseBool, so values parseBool accepts do not enable TLS here.
			name:   "capitalised true does not enable tls",
			config: map[string]string{splunkOtlpTlsKey: "True", splunkOtlpCaPemKey: "CA"},
			want:   base(),
		},
		{
			name:   "numeric one does not enable tls",
			config: map[string]string{splunkOtlpTlsKey: "1", splunkOtlpCaPemKey: "CA"},
			want:   base(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destConfig := map[string]string{splunkRealm: "eu0"}
			maps.Copy(destConfig, tt.config)

			cfg := spEmptyConfig()
			_, err := (&SplunkOTLP{}).ModifyConfig(spDest(destConfig), &cfg)
			require.NoError(t, err)

			assert.Equal(t, GenericMap{"otlp_http/" + spDestID: tt.want}, cfg.Exporters)
		})
	}
}

// TestSplunkRealmIsRequiredButNotValidated separates "the realm is absent" — which both
// configurers refuse — from "the realm is present and empty", which they accept and turn
// into an unreachable host.
func TestSplunkRealmIsRequiredButNotValidated(t *testing.T) {
	for name, configurer := range spConfigurers() {
		t.Run(name+"/absent realm writes nothing", func(t *testing.T) {
			cfg := spEmptyConfig()
			pipelines, err := configurer.ModifyConfig(spDest(map[string]string{}), &cfg)
			require.Error(t, err)
			assert.Empty(t, pipelines)
			// A refused destination must not leave a half-built exporter behind for the
			// gateway to load.
			assert.Empty(t, cfg.Exporters)
			assert.Empty(t, cfg.Service.Pipelines)
		})

		t.Run(name+"/empty realm is accepted", func(t *testing.T) {
			cfg := spEmptyConfig()
			_, err := configurer.ModifyConfig(spDest(map[string]string{splunkRealm: ""}), &cfg)
			// Only key presence is checked, so an empty realm produces a syntactically
			// valid endpoint pointing at a host that does not resolve. Pinned as current
			// behaviour, not endorsed.
			require.NoError(t, err)
			exporter := cfg.Exporters["otlp_http/"+spDestID].(GenericMap)
			assert.Equal(t, "https://ingest..signalfx.com/v2/trace/otlp", exporter["traces_endpoint"])
		})
	}
}

// TestSplunkWritesNothingWithoutTraces covers the signal gate for both configurers.
// The catalog declares traces as the only supported signal for either destination.
func TestSplunkWritesNothingWithoutTraces(t *testing.T) {
	for _, signal := range []common.ObservabilitySignal{common.MetricsObservabilitySignal, common.LogsObservabilitySignal} {
		for name, configurer := range spConfigurers() {
			t.Run(name+"/"+string(signal), func(t *testing.T) {
				cfg := spEmptyConfig()
				dest := &mockDestination{
					id:      spDestID,
					config:  map[string]string{splunkRealm: "eu0"},
					signals: []common.ObservabilitySignal{signal},
				}
				pipelines, err := configurer.ModifyConfig(dest, &cfg)
				require.NoError(t, err)
				assert.Empty(t, pipelines)
				assert.Empty(t, cfg.Exporters)
				assert.Empty(t, cfg.Service.Pipelines)
			})
		}
	}
}

// TestSplunkDestinationsDoNotShareAnExporter pins that the access token placeholder and
// the exporter name are both scoped by destination ID. Splunk is one of the few
// destinations whose exporter name carries no type prefix, so the ID is all that keeps
// two Splunk destinations — and their scoped tokens — apart.
func TestSplunkDestinationsDoNotShareAnExporter(t *testing.T) {
	cfg := spEmptyConfig()

	first := &mockDestination{id: "odigos.io.dest.splunk-aaaa", config: map[string]string{splunkRealm: "eu0"},
		signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}}
	second := &mockDestination{id: "odigos.io.dest.splunkotlp-bbbb", config: map[string]string{splunkRealm: "us1"},
		signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}}

	firstPipelines, err := (&Splunk{}).ModifyConfig(first, &cfg)
	require.NoError(t, err)
	secondPipelines, err := (&SplunkOTLP{}).ModifyConfig(second, &cfg)
	require.NoError(t, err)

	assert.Len(t, cfg.Exporters, 2)
	assert.Len(t, cfg.Service.Pipelines, 2)
	assert.NotEqual(t, secondPipelines[0], firstPipelines[0])

	firstExporter := cfg.Exporters["otlp_http/odigos.io.dest.splunk-aaaa"].(GenericMap)
	secondExporter := cfg.Exporters["otlp_http/odigos.io.dest.splunkotlp-bbbb"].(GenericMap)

	assert.Equal(t, GenericMap{"X-SF-Token": "${ODIGOS_DEST_odigos_io_dest_splunk_aaaa_SPLUNK_ACCESS_TOKEN}"}, firstExporter["headers"])
	assert.Equal(t, GenericMap{"X-SF-Token": "${ODIGOS_DEST_odigos_io_dest_splunkotlp_bbbb_SPLUNK_ACCESS_TOKEN}"}, secondExporter["headers"])
	assert.Equal(t, "https://ingest.eu0.signalfx.com/v2/trace/otlp", firstExporter["traces_endpoint"])
	assert.Equal(t, "https://ingest.us1.signalfx.com/v2/trace/otlp", secondExporter["traces_endpoint"])

	// Each pipeline must reference its own exporter, not the one written last.
	assert.Equal(t, []string{"otlp_http/odigos.io.dest.splunk-aaaa"}, cfg.Service.Pipelines[firstPipelines[0]].Exporters)
	assert.Equal(t, []string{"otlp_http/odigos.io.dest.splunkotlp-bbbb"}, cfg.Service.Pipelines[secondPipelines[0]].Exporters)
}
