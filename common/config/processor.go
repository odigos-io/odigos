package config

import (
	"fmt"
)

func CrdProcessorToConfig(processors []ProcessorConfigurer) (cfg Config,
	tracesProcessors []string, metricsProcessors []string, logsProcessors []string, errs map[string]error) {
	errs = make(map[string]error)
	processorsMap := GenericMap{}
	for _, processor := range processors {
		processorKey := fmt.Sprintf("%s/%s", processor.GetType(), processor.GetID())
		processorsConfig, err := processor.GetConfig()
		if err != nil {
			// TODO: write the error to the status of the processor
			// consider how to handle this error
			errs[processor.GetID()] = fmt.Errorf("failed to convert processor %q to collector config: %w", processor.GetID(), err)
			continue
		}
		if processorKey == "" || processorsConfig == nil {
			continue
		}
		processorsMap[processorKey] = processorsConfig

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
	cfg = Config{
		Processors: processorsMap,
	}
	if len(errs) != 0 {
		return cfg, tracesProcessors, metricsProcessors, logsProcessors, errs
	}

	return cfg, tracesProcessors, metricsProcessors, logsProcessors, errs
}
