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
	ProfilesProcessors              []string
	Errs                            map[string]error
}

// profilesCapableProcessorTypes are the OTel processor types proven to support the experimental
// profiles signal in the pinned collector build. User Actions that select the PROFILES signal are
// only wired into the profiles pipeline when they map to one of these types:
//   - resource  (AddClusterInfo)
//   - transform (RenameAttribute, DeleteAttribute)
//
// Types not listed here are intentionally excluded so a PROFILES selection can never produce an
// invalid collector config: k8sattributes is already applied unconditionally on the node profiles
// pipeline (see ProfilingPipelineConfig), redaction has no profiles consumer in this build, and
// span-only processors (span renamer, URL templatization, sampling) have no profiles analog.
var profilesCapableProcessorTypes = map[string]struct{}{
	"resource":  {},
	"transform": {},
}

func CrdProcessorToConfig(processors []ProcessorConfigurer) CrdProcessorResults {
	results := CrdProcessorResults{
		ProcessorsConfig: Config{
			Processors: GenericMap{},
		},
		TracesProcessorsPostSpanMetrics: []string{},
		MetricsProcessors:               []string{},
		LogsProcessors:                  []string{},
		ProfilesProcessors:              []string{},
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
		// Profiles is an experimental signal: only a subset of processor types can consume it.
		// Gate by type so selecting PROFILES on an unsupported action is a no-op rather than a
		// pipeline that fails to start.
		if isProfilingEnabled(processor) {
			if _, ok := profilesCapableProcessorTypes[processor.GetType()]; ok {
				results.ProfilesProcessors = append(results.ProfilesProcessors, processorKey)
			}
		}
	}
	if len(results.Errs) != 0 {
		return results
	}

	return results
}
