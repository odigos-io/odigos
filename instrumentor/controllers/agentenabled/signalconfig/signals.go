package signalconfig

import (
	"slices"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func GetEnabledSignalsForContainer(nodeCollectorsGroup *odigosv1.CollectorsGroup, irls *[]odigosv1.InstrumentationRule) (tracesEnabled bool, metricsEnabled bool, logsEnabled bool) {
	tracesEnabled = false
	metricsEnabled = false
	logsEnabled = false

	if nodeCollectorsGroup == nil {
		// if the node collectors group is not created yet,
		// it means the collectors are not running thus all signals are disabled.
		return false, false, false
	}

	// first set each signal to enabled/disabled based on the node collectors group global signals for collection.
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.TracesObservabilitySignal) {
		tracesEnabled = true
	}
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.MetricsObservabilitySignal) {
		metricsEnabled = true
	}
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.LogsObservabilitySignal) {
		logsEnabled = true
	}

	// disable specific signals if they are disabled in any of the workload level instrumentation rules.
	for _, irl := range *irls {

		// these signals are in the workload level,
		// and library specific rules are not relevant to the current calculation.
		if irl.Spec.InstrumentationLibraries != nil {
			continue
		}

		// if any instrumentation rule has trace config disabled, we should disable traces for this container.
		// the list is already filtered to only include rules that are relevant to the current workload.
		if irl.Spec.TraceConfig != nil && irl.Spec.TraceConfig.Disabled != nil && *irl.Spec.TraceConfig.Disabled {
			tracesEnabled = false
		}
	}

	return tracesEnabled, metricsEnabled, logsEnabled
}
