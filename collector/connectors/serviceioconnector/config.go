package serviceioconnector

import (
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
)

const defaultMetricsFlushInterval = 60 * time.Second

// Config configures the serviceio connector.
type Config struct {
	// InputSpanAttributes lists OpenTelemetry span attribute names read from inbound
	// spans when correlating service I/O on the receiving side (e.g. server spans).
	//
	// Each name should refer to a low-cardinality attribute (e.g. http.route,
	// rpc.service, db.system). More attributes produce finer-grained correlations
	// but increase memory, storage, and query cost.
	InputSpanAttributes []string `mapstructure:"input_span_attributes"`

	// OutputSpanAttributes lists OpenTelemetry span attribute names read from outbound
	// spans when correlating service I/O on the sending side (e.g. client/producer
	// spans and spans with cross-service children).
	//
	// Each name should refer to a low-cardinality attribute. More attributes produce
	// finer-grained correlations but increase memory, storage, and query cost.
	OutputSpanAttributes []string `mapstructure:"output_span_attributes"`

	// MetricsFlushInterval is the interval at which metrics are flushed to the exporter.
	// If set to 0, metrics are flushed on every received batch of traces.
	// Default is 60s if unset.
	MetricsFlushInterval *time.Duration `mapstructure:"metrics_flush_interval"`

	// MetricsTimestampOffset is the offset subtracted from metric export timestamps.
	// By default (0), data points are timestamped at flush time. Service I/O is derived
	// from complete traces that already waited in groupbytrace and may be aggregated
	// until the next flush, so exported metrics can appear later than the spans they
	// describe. A positive offset shifts timestamps backward (now - offset) to better
	// align with when the connection happened in the trace. This is a coarse global
	// adjustment, not per-trace timing; leave at 0 unless you need that alignment.
	MetricsTimestampOffset time.Duration `mapstructure:"metrics_timestamp_offset"`

	// OdigosConfigExtension references the odigos_config_k8s extension (implements OdigosConfigExtension).
	// When set, only service instances whose root span resource is an active Odigos source are counted.
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`
}

func (c *Config) Validate() error {
	if err := validateSpanAttributes("input_span_attributes", c.InputSpanAttributes); err != nil {
		return err
	}
	if err := validateSpanAttributes("output_span_attributes", c.OutputSpanAttributes); err != nil {
		return err
	}
	if c.OdigosConfigExtension != nil {
		typeStr := c.OdigosConfigExtension.Type().String()
		if _, err := component.NewType(typeStr); err != nil {
			return fmt.Errorf("invalid odigos_config_extension type %q: %w", typeStr, err)
		}
	}
	return nil
}

func validateSpanAttributes(fieldName string, keys []string) error {
	seen := make(map[string]struct{}, len(keys))
	for i, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			return fmt.Errorf("%s[%d] must not be empty", fieldName, i)
		}
		if _, exists := seen[trimmed]; exists {
			return fmt.Errorf("%s contains duplicate key %q", fieldName, trimmed)
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}

func normalizeSpanAttributes(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(keys))
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func (c *Config) resolvedMetricsFlushInterval() time.Duration {
	if c.MetricsFlushInterval == nil {
		return defaultMetricsFlushInterval
	}
	return *c.MetricsFlushInterval
}
