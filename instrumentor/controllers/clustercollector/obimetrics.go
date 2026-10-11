package clustercollector

import (
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
)

// obiMetricsRenameProcessorName renames OBI-produced metrics (e.g. network flow and TCP stats
// metrics named "obi.*" / "obi_*") by replacing the "obi" prefix with "odigos", for consistency
// and discoverability alongside other Odigos metrics in the platform.
const obiMetricsRenameProcessorName = "transform/odigos-obi-metrics-rename"

const resourceOdigosVersionProcessorName = "resource/odigos-version"

// obiMetricsRenameProcessorConfig returns a transform processor that replaces the "obi" name prefix of
// OBI metrics with "odigos". OBI emits metrics in dotted OTLP form (e.g. "obi.network.flow.bytes"),
// which becomes "odigos.network.flow.bytes"; the Prometheus underscore form ("obi_...") is handled
// defensively and becomes "odigos_...". The separator following the prefix is preserved by only
// replacing the leading "obi" token.
func obiMetricsRenameProcessorConfig() config.GenericMap {
	return config.GenericMap{
		"error_mode": "ignore",
		"metric_statements": []config.GenericMap{
			{
				"context": "metric",
				"statements": []string{
					`replace_pattern(name, "^obi", "odigos") where IsMatch(name, "^obi[._]")`,
				},
			},
		},
	}
}

// addObiMetricsRenamePipeline normalizes OBI metric names to the "odigos" prefix on the gateway's
// metrics root pipeline, before metrics are routed to destinations, so every destination sees the
// consistent Odigos naming. Running it on the gateway (rather than on each node collector) applies the
// rename once for the whole cluster. It is added unconditionally and is a no-op when no "obi.*"
// metrics are present. When metrics are not enabled on the gateway the metrics root pipeline is absent
// and this is a no-op.
func addObiMetricsRenamePipeline(c *config.Config) {
	metricsRootPipelineName := pipelinegen.GetTelemetryRootPipelineName(odigoscommon.MetricsObservabilitySignal)
	pipeline, exists := c.Service.Pipelines[metricsRootPipelineName]
	if !exists {
		return
	}

	if c.Processors == nil {
		c.Processors = make(config.GenericMap)
	}
	c.Processors[obiMetricsRenameProcessorName] = obiMetricsRenameProcessorConfig()

	// Run the rename early, before user (manifest) processors, so downstream processors and
	// destinations see the normalized names. The root pipeline starts with "resource/odigos-version";
	// insert the rename right after it when present.
	insertAt := 0
	if len(pipeline.Processors) > 0 && pipeline.Processors[0] == resourceOdigosVersionProcessorName {
		insertAt = 1
	}
	updated := make([]string, 0, len(pipeline.Processors)+1)
	updated = append(updated, pipeline.Processors[:insertAt]...)
	updated = append(updated, obiMetricsRenameProcessorName)
	updated = append(updated, pipeline.Processors[insertAt:]...)
	pipeline.Processors = updated
	c.Service.Pipelines[metricsRootPipelineName] = pipeline
}
