package common

import (
	"fmt"

	"encoding/json"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
)

func ProcessorCrToCollectorConfig(processor *odigosv1.Processor, collectorRole odigosv1.CollectorsGroupRole) (GenericMap, string, error) {

	// do not include disabled processors
	if processor.Spec.Disabled {
		return nil, "", nil
	}

	// ignore processors that do not participate in this collector role
	roleFound := false
	for _, role := range processor.Spec.CollectorRoles {
		if role == collectorRole {
			roleFound = true
		}
	}
	if !roleFound {
		return nil, "", nil
	}

	processorKey := fmt.Sprintf("%s/%s", processor.Spec.Type, processor.Name)
	var processorConfig map[string]interface{}
	err := json.Unmarshal(processor.Spec.Data.Raw, &processorConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal processor %s data: %v", processor.Name, err)
	}

	return processorConfig, processorKey, nil
}
