package clustercollector

import (
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/odigos-io/odigos/common/urltemplate"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
)

// effectiveCardinalityControl returns the cardinality-control config the gateway
// should be wired for. Live traffic learning of URL templatization rules is enterprise-only —
// helm already gates the helm value on an on-prem token, and community tier must
// not install the side-channel pipeline. Same reasoning as effectiveInsightsConfig.
func effectiveCardinalityControl(cc *common.CardinalityControlConfiguration, tier common.OdigosTier) *common.CardinalityControlConfiguration {
	if !tier.IsEnterprise() {
		return nil
	}
	return cc
}

// addUrlTemplatizationUnmatchedExporter appends the enterprise
// odigos_url_templatization exporter to the gateway root traces pipeline so
// every span fans out after root processors (including odigosurltemplate when
// that processor is configured). The exporter counts unmatched HTTP paths in
// cacheDb Redis. Noop when disabled or when no root traces pipeline exists.
func addUrlTemplatizationUnmatchedExporter(c *config.Config, odigosNs string, cardinalityControl *common.CardinalityControlConfiguration) error {
	if !common.UrlTemplatizationLiveTrafficLearningActive(cardinalityControl) {
		return nil
	}

	rootPipelineName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	rootPipeline, hasRoot := c.Service.Pipelines[rootPipelineName]
	if !hasRoot {
		return nil
	}

	if c.Exporters == nil {
		c.Exporters = config.GenericMap{}
	}

	c.Exporters[pipeline.UrlTemplatizationExporter] = config.GenericMap{
		"odigos_config_extension": k8sconsts.OdigosConfigK8sExtensionType,
		"endpoint":                k8sconsts.OdigosCacheEndpoint(odigosNs),
		"max_paths_per_workload":  maxExamplePathsPerWorkload(cardinalityControl),
		"path_idle_ttl":           pathExampleIdleTTL(cardinalityControl),
	}

	rootPipeline.Exporters = append(rootPipeline.Exporters, pipeline.UrlTemplatizationExporter)
	c.Service.Pipelines[rootPipelineName] = rootPipeline

	return nil
}

func liveTrafficLearningConfig(cardinalityControl *common.CardinalityControlConfiguration) *common.LiveTrafficLearningConfiguration {
	if cardinalityControl == nil ||
		cardinalityControl.UrlTemplatization == nil {
		return nil
	}
	return cardinalityControl.UrlTemplatization.LiveTrafficLearning
}

func maxExamplePathsPerWorkload(cardinalityControl *common.CardinalityControlConfiguration) int {
	ltl := liveTrafficLearningConfig(cardinalityControl)
	if ltl == nil {
		return urltemplate.ResolveMaxExamplePathsPerWorkload(nil)
	}
	return urltemplate.ResolveMaxExamplePathsPerWorkload(ltl.MaxExamplePathsPerWorkload)
}

func pathExampleIdleTTL(cardinalityControl *common.CardinalityControlConfiguration) string {
	ltl := liveTrafficLearningConfig(cardinalityControl)
	raw := ""
	if ltl != nil {
		raw = ltl.PathExampleIdleTTL
	}
	d := urltemplate.ResolvePathExampleIdleTTL(raw)
	if parsed, err := time.ParseDuration(raw); err == nil && parsed == d && parsed > 0 {
		return raw
	}
	return "48h"
}
