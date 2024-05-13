package common

import (
	"fmt"
	"sort"

	"encoding/json"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func IsProcessorTracingEnabled(processor common.ProcessorConfigurer) bool {
	for _, signal := range processor.GetSignals() {
		if signal == common.TracesObservabilitySignal {
			return true
		}
	}
	return false
}

func IsProcessorMetricsEnabled(processor common.ProcessorConfigurer) bool {
	for _, signal := range processor.GetSignals() {
		if signal == common.MetricsObservabilitySignal {
			return true
		}
	}
	return false
}

func IsProcessorLogsEnabled(processor common.ProcessorConfigurer) bool {
	for _, signal := range processor.GetSignals() {
		if signal == common.LogsObservabilitySignal {
			return true
		}
	}
	return false
}

func FilterAndSortProcessorsByOrderHint(processors *odigosv1.ProcessorList, collectorRole odigosv1.CollectorsGroupRole) []*odigosv1.Processor {
	filteredProcessors := []*odigosv1.Processor{}
	for i, processor := range processors.Items {

		// do not include disabled processors
		if processor.Spec.Disabled {
			continue
		}

		// take only processors that participate in this collector role
		for _, role := range processor.Spec.CollectorRoles {
			if role == collectorRole {
				filteredProcessors = append(filteredProcessors, &processors.Items[i])
			}
		}
	}

	// Now sort the filteredProcessors by the OrderHint property
	sort.Slice(filteredProcessors, func(i, j int) bool {
		return filteredProcessors[i].Spec.OrderHint < filteredProcessors[j].Spec.OrderHint
	})

	return filteredProcessors
}

func GetCrdProcessorsConfigMap(processors []common.ProcessorConfigurer) (cfg GenericMap, tracesProcessors []string, metricsProcessors []string, logsProcessors []string) {
	cfg = GenericMap{}
	for _, processor := range processors {
		fmt.Printf("processors: %+v\n", processor)
		processorKey := fmt.Sprintf("%s/%s", processor.GetType(), processor.GetName())
		processorsConfig, err := processor.GetConfig()
		fmt.Printf("err: %+v\n", err)
		if err != nil {
			// TODO: write the error to the status of the processor
			// consider how to handle this error
			log.Log.V(0).Info("failed to convert data-collection processor to collector config", "processor", processor.GetName(), "error", err)
			continue
		}
		if processorKey == "" || processorsConfig == nil {
			continue
		}
		cfg[processorKey] = processorsConfig

		if IsProcessorTracingEnabled(processor) {
			tracesProcessors = append(tracesProcessors, processorKey)
		}
		if IsProcessorMetricsEnabled(processor) {
			metricsProcessors = append(metricsProcessors, processorKey)
		}
		if IsProcessorLogsEnabled(processor) {
			logsProcessors = append(logsProcessors, processorKey)
		}
	}
	return
}
