package signals

import (
	"slices"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

type EnabledSignals struct {
	TracesEnabled  bool
	MetricsEnabled bool
	LogsEnabled    bool
}

func GetEnabledSignalsForContainer(nodeCollectorsGroup *odigosv1.CollectorsGroup, irls *[]odigosv1.InstrumentationRule) (EnabledSignals, *odigosv1.AgentDisabledInfo) {

	enabledSignals := EnabledSignals{
		TracesEnabled:  false,
		MetricsEnabled: false,
		LogsEnabled:    false,
	}

	if nodeCollectorsGroup == nil {
		// if the node collectors group is not created yet,
		// it means the collectors are not running thus all signals are disabled.
		return enabledSignals, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonWaitingForNodeCollector,
			AgentEnabledMessage: "waiting for OpenTelemetry Collector to be created",
		}
	}

	// first set each signal to enabled/disabled based on the node collectors group global signals for collection.
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.TracesObservabilitySignal) {
		enabledSignals.TracesEnabled = true
	}
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.MetricsObservabilitySignal) {
		enabledSignals.MetricsEnabled = true
	}
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.LogsObservabilitySignal) {
		enabledSignals.LogsEnabled = true
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
			enabledSignals.TracesEnabled = false
		}
	}

	if !enabledSignals.TracesEnabled && !enabledSignals.MetricsEnabled && !enabledSignals.LogsEnabled {
		return enabledSignals, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonNoCollectedSignals,
			AgentEnabledMessage: "all signals are disabled, no agent will be injected",
		}
	}

	return enabledSignals, nil
}
