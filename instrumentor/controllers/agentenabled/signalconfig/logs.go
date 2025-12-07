package signalconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func CalculateLogsConfig(logsEnabled bool, effectiveConfig *common.OdigosConfiguration, containerName string) (*odigosv1.AgentLogsConfig, *odigosv1.ContainerAgentConfig) {
	if !logsEnabled {
		return nil, nil
	}

	return &odigosv1.AgentLogsConfig{}, nil
}
