package profiles

import "github.com/odigos-io/odigos/common"

func modifySizingConfig(c *common.OdigosConfiguration, clusterCollectorConfig common.CollectorGatewayConfiguration, nodeCollectorConfig common.CollectorNodeConfiguration) {
	// do not modify the configuration if any of the values if they are already set
	if c.CollectorGateway != nil {
		return
	}
	// the following is not very elegant.
	// we only care if the sizing parameters are set, if the port is set, we apply it nevertheless
	if c.CollectorNode != nil && (c.CollectorNode.RequestMemoryMiB != 0 || c.CollectorNode.LimitMemoryMiB != 0 || c.CollectorNode.RequestCPUm != 0 || c.CollectorNode.LimitCPUm != 0) {
		return
	}

	c.CollectorGateway = &clusterCollectorConfig
	collectorNodeConfig := nodeCollectorConfig
	if c.CollectorNode != nil {
		// make sure we keep the port which is unrelated to the sizing
		collectorNodeConfig.CollectorOwnMetricsPort = c.CollectorNode.CollectorOwnMetricsPort
	}
	c.CollectorNode = &collectorNodeConfig
}
