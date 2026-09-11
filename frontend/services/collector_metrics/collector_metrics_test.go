package collectormetrics

import "testing"

func TestNormalizeTrafficMetricName(t *testing.T) {
	tests := map[string]string{
		traceSizeMetricName:     traceSizeMetricName,
		metricSizeMetricName:    metricSizeMetricName,
		logSizeMetricName:       logSizeMetricName,
		traceSizeMetricNameRaw:  traceSizeMetricName,
		metricSizeMetricNameRaw: metricSizeMetricName,
		logSizeMetricNameRaw:    logSizeMetricName,
		"unrelated_metric_name": "unrelated_metric_name",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			if actual := normalizeTrafficMetricName(input); actual != expected {
				t.Fatalf("normalizeTrafficMetricName(%q) = %q, want %q", input, actual, expected)
			}
		})
	}
}
