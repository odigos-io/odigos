package clustercollector

import (
	"fmt"
	"slices"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
)

const gatewayProfilesPipeline = "profiles"

func addInterrogationExporters(c *config.Config, namespace string, interrogation *common.InterrogationConfiguration) {
	if !common.InterrogationActive(interrogation) {
		return
	}
	profilesPipeline, hasProfiles := c.Service.Pipelines[gatewayProfilesPipeline]
	rootName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	rootPipeline, hasTraces := c.Service.Pipelines[rootName]
	if !hasProfiles && !hasTraces {
		return
	}
	if c.Extensions == nil {
		c.Extensions = config.GenericMap{}
	}
	c.Extensions[commonconf.InterrogationCacheExtension] = config.GenericMap{}
	if !slices.Contains(c.Service.Extensions, commonconf.InterrogationCacheExtension) {
		c.Service.Extensions = append(c.Service.Extensions, commonconf.InterrogationCacheExtension)
	}
	if c.Exporters == nil {
		c.Exporters = config.GenericMap{}
	}
	if hasProfiles {
		c.Exporters[commonconf.InterrogationProfilesExporter] = config.GenericMap{
			"interrogation_cache_extension": commonconf.InterrogationCacheExtension,
		}
		profilesPipeline.Exporters = append(profilesPipeline.Exporters, commonconf.InterrogationProfilesExporter)
		c.Service.Pipelines[gatewayProfilesPipeline] = profilesPipeline
	}
	if hasTraces {
		endpoint := interrogation.EvidenceEndpoint
		if endpoint == "" {
			endpoint = fmt.Sprintf("http://ui.%s.svc:3000/api/interrogation/evidence", namespace)
		}
		c.Exporters[commonconf.InterrogationTracesExporter] = config.GenericMap{
			"interrogation_cache_extension": commonconf.InterrogationCacheExtension,
			"evidence_endpoint":             endpoint,
		}
		rootPipeline.Exporters = append(rootPipeline.Exporters, commonconf.InterrogationTracesExporter)
		c.Service.Pipelines[rootName] = rootPipeline
	}
}
