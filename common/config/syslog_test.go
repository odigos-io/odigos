package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

func TestSyslogConfig(t *testing.T) {
	for _, tc := range []struct {
		name  string
		data  map[string]string
		fails bool
	}{
		{"tls defaults", map[string]string{}, false},
		{"plaintext", map[string]string{"SYSLOG_TLS_ENABLED": "false"}, false},
		{"udp", map[string]string{"SYSLOG_NETWORK": "udp", "SYSLOG_TLS_ENABLED": "false"}, false},
		{"rfc3164", map[string]string{"SYSLOG_PROTOCOL": "rfc3164"}, false},
		{"mutual tls", map[string]string{"SYSLOG_MTLS_ENABLED": "true", "SYSLOG_TLS_ENABLED": "false"}, false},
		{"ipv6", map[string]string{"SYSLOG_ENDPOINT": "::1"}, false},
		{"missing host", map[string]string{"SYSLOG_ENDPOINT": ""}, true},
		{"url", map[string]string{"SYSLOG_ENDPOINT": "https://example.com"}, true},
		{"host with port", map[string]string{"SYSLOG_ENDPOINT": "example.com:514"}, true},
		{"port nonnumeric", map[string]string{"SYSLOG_PORT": "bad"}, true},
		{"port out of range", map[string]string{"SYSLOG_PORT": "65536"}, true},
		{"unknown network", map[string]string{"SYSLOG_NETWORK": "unix"}, true},
		{"unknown protocol", map[string]string{"SYSLOG_PROTOCOL": "cef"}, true},
		{"udp tls", map[string]string{"SYSLOG_NETWORK": "udp"}, true},
		{"udp mtls", map[string]string{"SYSLOG_NETWORK": "udp", "SYSLOG_TLS_ENABLED": "false", "SYSLOG_MTLS_ENABLED": "true"}, true},
		{"rfc3164 octets", map[string]string{"SYSLOG_PROTOCOL": "rfc3164", "SYSLOG_OCTET_COUNTING": "true"}, true},
		{"invalid framing", map[string]string{"SYSLOG_OCTET_COUNTING": "invalid"}, true},
		{"facility", map[string]string{"SYSLOG_FACILITY": "24"}, true},
		{"oversize", map[string]string{"SYSLOG_MAX_MESSAGE_BYTES": "64001"}, true},
		{"rfc3164 oversize", map[string]string{"SYSLOG_PROTOCOL": "rfc3164", "SYSLOG_MAX_MESSAGE_BYTES": "481"}, true},
		{"zero size", map[string]string{"SYSLOG_MAX_MESSAGE_BYTES": "0"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]string{"SYSLOG_ENDPOINT": "syslog.example.com"}
			for k, v := range tc.data {
				data[k] = v
			}
			dest := rawDataDestination{ID: "odigos.io.dest.syslog-test", Type: common.SyslogDestinationType, Data: data, Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal}}
			cfg := &config.Config{Exporters: config.GenericMap{}, Processors: config.GenericMap{}, Service: config.Service{Pipelines: map[string]config.Pipeline{}}}
			names, err := (&config.Syslog{}).ModifyConfig(dest, cfg)
			if tc.fails {
				require.Error(t, err)
				require.Empty(t, cfg.Exporters)
				require.Empty(t, cfg.Processors)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{"logs/syslog-" + dest.ID}, names)
			exp := cfg.Exporters["syslog/"+dest.ID].(config.GenericMap)
			tls := exp["tls"].(config.GenericMap)
			if data["SYSLOG_MTLS_ENABLED"] == "true" {
				require.Equal(t, false, tls["insecure"])
				require.Equal(t, config.SecretEnvPlaceholder("SYSLOG_CLIENT_KEY_PEM", dest), tls["key_pem"])
				require.NotContains(t, tls, "insecure_skip_verify")
			}
			if tc.name == "tls defaults" {
				require.Equal(t, 6514, exp["port"])
				require.Equal(t, true, exp["enable_octet_counting"])
			}
			if tc.name == "plaintext" || tc.name == "udp" {
				require.Equal(t, 514, exp["port"])
				require.Equal(t, true, tls["insecure"])
			}
			if tc.name == "ipv6" {
				require.Equal(t, "[::1]", exp["endpoint"])
			}
		})
	}
}

func TestSyslogRejectsOtherSignals(t *testing.T) {
	for _, signal := range []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal} {
		dest := rawDataDestination{ID: "test", Type: common.SyslogDestinationType, Data: map[string]string{"SYSLOG_ENDPOINT": "localhost"}, Signals: []common.ObservabilitySignal{common.LogsObservabilitySignal, signal}}
		_, err := (&config.Syslog{}).ModifyConfig(dest, &config.Config{})
		require.Error(t, err)
	}
}
