package common

import (
	"fmt"
	"sort"

	"encoding/json"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
)

func IsProcessorTracingEnabled(processor *odigosv1.Processor) bool {
	for _, signal := range processor.Spec.Signals {
		if signal == common.TracesObservabilitySignal {
			return true
		}
	}
	return false
}

func IsProcessorMetricsEnabled(processor *odigosv1.Processor) bool {
	for _, signal := range processor.Spec.Signals {
		if signal == common.MetricsObservabilitySignal {
			return true
		}
	}
	return false
}

func IsProcessorLogsEnabled(processor *odigosv1.Processor) bool {
	for _, signal := range processor.Spec.Signals {
		if signal == common.LogsObservabilitySignal {
			return true
		}
	}
	return false
}

func FilterAndSortProcessorsByOrderHint(processors *odigosv1.ProcessorList, collectorRole odigosv1.CollectorsGroupRole) []*odigosv1.Processor {

	filteredProcessors := []*odigosv1.Processor{}
	for _, processor := range processors.Items {

		// do not include disabled processors
		if processor.Spec.Disabled {
			continue
		}

		// take only processors that participate in this collector role
		for _, role := range processor.Spec.CollectorRoles {
			if role == collectorRole {
				filteredProcessors = append(filteredProcessors, &processor)
			}
		}
	}

	// Now sort the filteredProcessors by the OrderHint property
	sort.Slice(filteredProcessors, func(i, j int) bool {
		return filteredProcessors[i].Spec.OrderHint < filteredProcessors[j].Spec.OrderHint
	})

	return filteredProcessors
}

func ProcessorCrToCollectorConfig(processor *odigosv1.Processor, collectorRole odigosv1.CollectorsGroupRole) (GenericMap, string, error) {
	processorKey := fmt.Sprintf("%s/%s", processor.Spec.Type, processor.Name)
	var processorConfig map[string]interface{}
	err := json.Unmarshal(processor.Spec.Data.Raw, &processorConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal processor %s data: %v", processor.Name, err)
	}

	return processorConfig, processorKey, nil
}
