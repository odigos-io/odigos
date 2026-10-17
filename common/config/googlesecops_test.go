package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestGoogleSecOpsConfig(t *testing.T) {
	fields := map[string]string{
		"GOOGLE_SECOPS_PROJECT":     "123456789012",
		"GOOGLE_SECOPS_CUSTOMER_ID": "00000000-0000-0000-0000-000000000001",
		"GOOGLE_SECOPS_LOCATION":    "europe",
		"GOOGLE_SECOPS_LOG_TYPE":    "ODIGOS_APPLICATION",
	}
	dest := &mockDestination{id: "odigos.io.dest.googlesecops-one", config: fields}
	cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	pipelines, err := (&GoogleSecOps{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"logs/googlesecops-" + dest.id}, pipelines)
	require.Len(t, cfg.Service.Pipelines, 1)
	exporter := cfg.Exporters["chronicle/"+dest.id].(GenericMap)
	require.Equal(t, "https", exporter["protocol"])
	require.Equal(t, "v1", exporter["api_version"])
	require.Equal(t, "europe", exporter["location"])
	require.Equal(t, "chronicle.googleapis.com", exporter["endpoint"])
	require.Equal(t, false, exporter["override_endpoint"])
	require.Equal(t, "gzip", exporter["compression"])
	require.Equal(t, "${ODIGOS_DEST_odigos_io_dest_googlesecops_one_GOOGLE_SECOPS_CREDENTIALS}", exporter["creds"])
	require.Equal(t, GenericMap{"enabled": true}, exporter["retry_on_failure"])
	require.NotContains(t, exporter, "raw_log_field")

	fields["GOOGLE_SECOPS_ENDPOINT"] = "chronicle.europe.rep.googleapis.com"
	fields["GOOGLE_SECOPS_NAMESPACE"] = "production"
	_, err = (&GoogleSecOps{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	exporter = cfg.Exporters["chronicle/"+dest.id].(GenericMap)
	require.Equal(t, fields["GOOGLE_SECOPS_ENDPOINT"], exporter["endpoint"])
	require.Equal(t, true, exporter["override_endpoint"])
	require.Equal(t, "production", exporter["namespace"])

	for _, key := range []string{"GOOGLE_SECOPS_PROJECT", "GOOGLE_SECOPS_CUSTOMER_ID", "GOOGLE_SECOPS_LOCATION", "GOOGLE_SECOPS_LOG_TYPE"} {
		t.Run(key, func(t *testing.T) {
			value := fields[key]
			fields[key] = " "
			invalid := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			_, err := (&GoogleSecOps{}).ModifyConfig(dest, invalid)
			require.ErrorContains(t, err, key)
			require.Empty(t, invalid.Exporters)
			fields[key] = value
		})
	}
	fields["GOOGLE_SECOPS_ENDPOINT"] = "https://chronicle.googleapis.com"
	_, err = (&GoogleSecOps{}).ModifyConfig(dest, cfg)
	require.ErrorContains(t, err, "hostname")
	dest.signals = []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.ProfilesObservabilitySignal}
	_, err = (&GoogleSecOps{}).ModifyConfig(dest, cfg)
	require.ErrorContains(t, err, "logs")
}
