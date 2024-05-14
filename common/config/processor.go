package config

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
			log.Log.V(0).Info("failed to convert processor to collector config", "processor", processor.GetName(), "error", err)
			continue
		}
		if processorKey == "" || processorsConfig == nil {
			continue
		}
		cfg[processorKey] = processorsConfig

		if isTracingEnabled(processor) {
			tracesProcessors = append(tracesProcessors, processorKey)
		}
		if isMetricsEnabled(processor) {
			metricsProcessors = append(metricsProcessors, processorKey)
		}
		if isLoggingEnabled(processor) {
			logsProcessors = append(logsProcessors, processorKey)
		}
	}
	return
}
