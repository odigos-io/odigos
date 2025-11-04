package config

import (
	"fmt"
)

type CrdProcessorResults struct {
	ProcessorsConfig                Config
	TracesProcessors                []string
	TracesProcessorsPostSpanMetrics []string
	MetricsProcessors               []string
	LogsProcessors                  []string
	Errs                            map[string]error
}

func CrdProcessorToConfig(processors []ProcessorConfigurer) CrdProcessorResults {
	results := CrdProcessorResults{
		ProcessorsConfig: Config{
			Processors: GenericMap{},
		},
		TracesProcessorsPostSpanMetrics: []string{},
		MetricsProcessors:               []string{},
		LogsProcessors:                  []string{},
		Errs:                            make(map[string]error),
	}

	for _, processor := range processors {
		processorKey := fmt.Sprintf("%s/%s", processor.GetType(), processor.GetID())
		processorsConfig, err := processor.GetConfig()
		if err != nil {
			// TODO: write the error to the status of the processor
			// consider how to handle this error
			results.Errs[processor.GetID()] = fmt.Errorf("failed to convert processor %q to collector config: %w", processor.GetID(), err)
			continue
		}
		if processorKey == "" || processorsConfig == nil {
			continue
		}
		results.ProcessorsConfig.Processors[processorKey] = processorsConfig

		if isTracingEnabled(processor) {
			// for traces processors, we differentiate between 2:
			// - regular ones with order hint < 10
			// - those that have order hint >= 10, which are applied for exporting, but after spanmetrics is calculated.
			// it can be used to add simple sampling (not tail) in node-collector, which will happen after the span metrics are calculated.
			if processor.GetOrderHint() < 10 {
				results.TracesProcessors = append(results.TracesProcessors, processorKey)
			} else {
				results.TracesProcessorsPostSpanMetrics = append(results.TracesProcessorsPostSpanMetrics, processorKey)
			}
		}
		if isMetricsEnabled(processor) {
			results.MetricsProcessors = append(results.MetricsProcessors, processorKey)
		}
		if isLoggingEnabled(processor) {
			results.LogsProcessors = append(results.LogsProcessors, processorKey)
		}
	}
	if len(results.Errs) != 0 {
		return results
	}

	return results
}
