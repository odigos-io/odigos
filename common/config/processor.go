package config

import (
	"fmt"
)

func CrdProcessorToConfig(processors []ProcessorConfigurer) (cfg Config,
	tracesProcessors []string, tracesProcessorsPostSpanMetrics []string, metricsProcessors []string, logsProcessors []string, errs map[string]error) {
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
			// for traces processors, we differentiate between 2:
			// - regular ones with order hint < 10
			// - those that have order hint >= 10, which are applied for exporting, but after spanmetrics is calculated.
			// it can be used to add simple sampling (not tail) in node-collector, which will happen after the span metrics are calculated.
			if processor.GetOrderHint() < 10 {
				tracesProcessors = append(tracesProcessors, processorKey)
			} else {
				tracesProcessorsPostSpanMetrics = append(tracesProcessorsPostSpanMetrics, processorKey)
			}
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
		return cfg, tracesProcessors, tracesProcessorsPostSpanMetrics, metricsProcessors, logsProcessors, errs
	}

	return cfg, tracesProcessors, tracesProcessorsPostSpanMetrics, metricsProcessors, logsProcessors, errs
}
