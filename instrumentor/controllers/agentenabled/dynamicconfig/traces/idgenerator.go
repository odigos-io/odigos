package traces

import (
	"fmt"
	"strconv"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func TimedWallEnabled(effectiveConfig *common.OdigosConfiguration) bool {
	return effectiveConfig.TraceIdSuffix != ""
}

func CalculateIdGeneratorConfig(effectiveConfig *common.OdigosConfiguration) (*odigosv1.IdGeneratorConfig, *odigosv1.AgentDisabledInfo) {

	// currentlly supporting just one id generator type
	if !TimedWallEnabled(effectiveConfig) {
		return nil, nil
	}

	sourceId, err := strconv.ParseUint(effectiveConfig.TraceIdSuffix, 16, 8)
	if err != nil {
		return nil, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
			AgentEnabledMessage: fmt.Sprintf("failed to parse trace id suffix: %s. trace id suffix must be a single byte hex value (for example 'A3')", err),
		}
	}

	return &odigosv1.IdGeneratorConfig{
		TimedWall: &odigosv1.IdGeneratorTimedWallConfig{
			SourceId: uint8(sourceId),
		},
	}, nil
}
