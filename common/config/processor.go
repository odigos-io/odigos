package config

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func IsProcessorTracingEnabled(processor ProcessorConfigurer) bool {
	for _, signal := range processor.GetSignals() {
		if signal == common.TracesObservabilitySignal {
			return true
		}
	}
	return false
}

func IsProcessorMetricsEnabled(processor ProcessorConfigurer) bool {
	for _, signal := range processor.GetSignals() {
		if signal == common.MetricsObservabilitySignal {
			return true
		}
	}
	return false
}

func IsProcessorLogsEnabled(processor ProcessorConfigurer) bool {
	for _, signal := range processor.GetSignals() {
		if signal == common.LogsObservabilitySignal {
			return true
		}
	}
	return false
}

func GetCrdProcessorsConfigMap(processors []ProcessorConfigurer) (cfg GenericMap, tracesProcessors []string, metricsProcessors []string, logsProcessors []string) {
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
