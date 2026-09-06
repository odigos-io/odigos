package common

import (
	"encoding/json"
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	actionutil "github.com/odigos-io/odigos/k8sutils/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func ConvertActionsToConfigExtensionProcessors(actions odigosv1.ActionList) []odigosv1.Processor {
	processorToSignals := aggregateConfigExtensionProcessorTypes(actions)
	processors := []odigosv1.Processor{}
	for processorType, signals := range processorToSignals {
		processors = append(processors, actionToConfigExtensionProcessor(processorType, signals))
	}
	return processors
}

// IsLegacyConfigExtensionProcessorType reports Processor CR types that used to back
// actions now applied via odigosConfigExtension.
//
// TODO(remove by 2027-01): temporary migration cleanup; delete once leftover Processor CRs
// from older Odigos versions are no longer expected in the field.
func IsLegacyConfigExtensionProcessorType(processorType string) bool {
	switch processorType {
	case consts.OdigosPiiMaskingProcessorType, consts.OdigosSQLQueryProcessorType:
		return true
	default:
		return false
	}
}

func aggregateConfigExtensionProcessorTypes(actions odigosv1.ActionList) map[string][]odigoscommon.ObservabilitySignal {

	// aggregate unique signals from all actions per processor type
	processorToSignals := map[string][]odigoscommon.ObservabilitySignal{}
	for i := range actions.Items {
		action := &actions.Items[i]
		processorTypes := extractConfigExtActions(action)
		// The Action CRD accepts any signal, but wiring a processor into a pipeline for a signal
		// it cannot consume makes the collector fail to start, so unsupported signals are dropped.
		supportedSignals := actionutil.SupportedSignals(action)
		for _, processorType := range processorTypes {
			for _, signal := range action.Spec.Signals {
				if !slices.Contains(supportedSignals, signal) {
					continue
				}
				// only if this signal is not already in the list for this processor type
				// the list is short (<5 items) so no problem iterating it
				if !slices.Contains(processorToSignals[processorType], signal) {
					processorToSignals[processorType] = append(processorToSignals[processorType], signal)
				}
			}
		}
	}

	return processorToSignals
}

func extractConfigExtActions(action *odigosv1.Action) []string {
	if action.Spec.Disabled {
		return []string{}
	}
	return actionutil.ConfigExtensionProcessorTypes(action)
}

func actionToConfigExtensionProcessor(processorType string, signals []odigoscommon.ObservabilitySignal) odigosv1.Processor {

	configJSON, _ := json.Marshal(map[string]interface{}{
		"odigos_config_extension": k8sconsts.OdigosConfigK8sExtensionType,
	})

	return odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Processor",
			APIVersion: "odigos.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: processorType,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            processorType,
			Signals:         signals,
			OrderHint:       0, // need to revisit
			CollectorRoles:  []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleClusterGateway},
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}
}
