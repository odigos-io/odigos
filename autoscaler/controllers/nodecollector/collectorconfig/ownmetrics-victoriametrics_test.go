package collectorconfig

import (
	"regexp"
	"testing"

	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// odigletScrapeKeepRule digs the single metric_relabel_config out of the built prometheus receiver.
func odigletScrapeKeepRule(t *testing.T) config.GenericMap {
	t.Helper()

	receiver, ok := odigletMetricsReceiverConfig()[odigletMetricsReceiverName].(config.GenericMap)
	require.True(t, ok, "odiglet metrics receiver is missing")
	receiverConfig, ok := receiver["config"].(config.GenericMap)
	require.True(t, ok, "receiver has no config")
	scrapeConfigs, ok := receiverConfig["scrape_configs"].([]config.GenericMap)
	require.True(t, ok, "receiver has no scrape_configs")
	require.Len(t, scrapeConfigs, 1)
	relabelConfigs, ok := scrapeConfigs[0]["metric_relabel_configs"].([]config.GenericMap)
	require.True(t, ok, "scrape config has no metric_relabel_configs")
	require.Len(t, relabelConfigs, 1)

	return relabelConfigs[0]
}

// The odiglet exposes the full process and collector metric surface, so everything Odigos does not
// explicitly keep here is dropped before it reaches the metrics store. A pattern that stops
// matching means the metric silently disappears from the UI rather than failing anywhere.
func TestOdigletMetricsScrapeKeepList(t *testing.T) {
	rule := odigletScrapeKeepRule(t)

	assert.Equal(t, "keep", rule["action"], "the rule must be a keep list, dropping it inverts the filter")
	assert.Equal(t, []string{"__name__"}, rule["source_labels"])

	rawRegex, ok := rule["regex"].(string)
	require.True(t, ok, "keep rule has no regex")
	// Prometheus anchors relabel regexes, so a pattern only keeps a metric when it matches the
	// whole name.
	keep, err := regexp.Compile("^(?:" + rawRegex + ")$")
	require.NoError(t, err, "the keep regex must be a valid prometheus regex")

	tests := []struct {
		metric string
		want   bool
	}{
		{metric: "odigos_java_ebpf_instrumentation_spans_total", want: true},
		{metric: "odigos_python_ebpf_instrumentation_spans_total", want: true},
		{metric: "odigos_nodejs_ebpf_instrumentation_spans_total", want: true},
		{metric: "odigos_ebpf_events_sent_go", want: true},
		{metric: "odigos_ebpf_events_send_failed_go", want: true},
		{metric: "odigos_ebpf_ring_pending_bytes", want: true},
		// A histogram reaches prometheus as three series, and all of them are needed to compute a
		// duration quantile.
		{metric: "odigos_ebpf_probe_handler_duration_us_microseconds_bucket", want: true},
		{metric: "odigos_ebpf_probe_handler_duration_us_microseconds_sum", want: true},
		{metric: "odigos_ebpf_probe_handler_duration_us_microseconds_count", want: true},

		// Only the three languages named in the instrumentation pattern are kept.
		{metric: "odigos_go_ebpf_instrumentation_spans_total", want: false},
		{metric: "odigos_ebpf_events_dropped_go", want: false},
		// The collector's own process metrics are reported through the own-metrics pipeline instead.
		{metric: "otelcol_process_cpu_seconds", want: false},
		{metric: "go_gc_duration_seconds", want: false},
		// Anchoring: a metric that merely contains a kept name is not kept.
		{metric: "test_odigos_ebpf_ring_pending_bytes", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			assert.Equal(t, tt.want, keep.MatchString(tt.metric))
		})
	}
}

// Scraped odiglet metrics leave the node collector over their own OTLP http exporter, and nothing
// fails when its address is wrong: the metrics just never arrive at the metrics store.
func TestOdigletMetricsExporterAddressesTheClusterCollector(t *testing.T) {
	got := OdigletMetricsConfig("custom-odigos-ns")

	exporter, ok := got.Exporters[odigletMetricsExporterName].(config.GenericMap)
	require.True(t, ok, "odiglet metrics exporter is missing")
	assert.Equal(t, "http://odigos-gateway.custom-odigos-ns:44318", exporter["endpoint"],
		"the exporter must address the cluster collector in the namespace Odigos is installed in")

	pipeline, ok := got.Service.Pipelines[odigletMetricsPipelineName]
	require.True(t, ok, "odiglet metrics pipeline is missing")
	assert.Equal(t, []string{odigletMetricsReceiverName}, pipeline.Receivers)
	assert.Equal(t, []string{odigletMetricsExporterName}, pipeline.Exporters)
	// Every node's data collection pod scrapes the same localhost target, so without stamping the
	// pod name the series from all of them collapse into one.
	assert.Equal(t, []string{"resource/odiglet-pod-name"}, pipeline.Processors)
	assert.Contains(t, got.Processors, "resource/odiglet-pod-name")
}
