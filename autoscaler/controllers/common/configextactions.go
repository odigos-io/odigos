package common

import (
	"encoding/json"
	"slices"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	actionutil "github.com/odigos-io/odigos/k8sutils/pkg/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func ConvertActionsToConfigExtensionProcessors(actionList odigosv1.ActionList, spanMetricsEnabled bool) []odigosv1.Processor {
	processorToSignals := aggregateConfigExtensionProcessorTypes(actionList)
	processors := []odigosv1.Processor{}
	for processorType, signals := range processorToSignals {
		processors = append(processors, actionToConfigExtensionProcessor(processorType, signals, spanMetricsEnabled))
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

func aggregateConfigExtensionProcessorTypes(actionList odigosv1.ActionList) map[string][]odigoscommon.ObservabilitySignal {

	// aggregate unique signals from all actions per processor type
	processorToSignals := map[string][]odigoscommon.ObservabilitySignal{}
	for i := range actionList.Items {
		action := &actionList.Items[i]
		processorTypes := extractConfigExtActions(action)
		for _, processorType := range processorTypes {
			for _, signal := range action.Spec.Signals {
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

func actionToConfigExtensionProcessor(processorType string, signals []odigoscommon.ObservabilitySignal, spanMetricsEnabled bool) odigosv1.Processor {

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
			OrderHint:       configExtensionProcessorOrderHint(processorType),
			CollectorRoles:  configExtensionProcessorCollectorRoles(processorType, spanMetricsEnabled),
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}
}

func configExtensionProcessorOrderHint(processorType string) int {
	switch processorType {
	case consts.OdigosSQLQueryProcessorType:
		// Match DbQueryTemplatization / InferDbAttributes: run before spanmetrics on the node collector.
		return actions.DbQueryTemplatizationConfig{}.OrderHint()
	case consts.OdigosPiiMaskingProcessorType:
		return actions.PiiMaskingConfig{}.OrderHint()
	default:
		return 0
	}
}

func configExtensionProcessorCollectorRoles(processorType string, spanMetricsEnabled bool) []odigosv1.CollectorsGroupRole {
	var roles []k8sconsts.CollectorRole
	switch processorType {
	case consts.OdigosSQLQueryProcessorType:
		// Span metrics are built in the node collector. SQL query processing must run there
		// first so span names / query attributes are normalized before cardinality is recorded.
		roles = actions.DbQueryTemplatizationConfig{}.SharedProcessorCollectorRoles(spanMetricsEnabled)
	case consts.OdigosPiiMaskingProcessorType:
		roles = actions.PiiMaskingConfig{}.CollectorRoles()
	default:
		roles = []k8sconsts.CollectorRole{k8sconsts.CollectorsRoleClusterGateway}
	}

	out := make([]odigosv1.CollectorsGroupRole, 0, len(roles))
	for _, r := range roles {
		out = append(out, odigosv1.CollectorsGroupRole(r))
	}
	return out
}

// ConfigExtensionActionCollectorRole returns which collector role currently hosts a
// config-extension action's processor, matching ConvertActionsToConfigExtensionProcessors.
func ConfigExtensionActionCollectorRole(action *odigosv1.Action, spanMetricsEnabled bool) odigosv1.CollectorsGroupRole {
	processorTypes := actionutil.ConfigExtensionProcessorTypes(action)
	for _, processorType := range processorTypes {
		roles := configExtensionProcessorCollectorRoles(processorType, spanMetricsEnabled)
		for _, role := range roles {
			if role == odigosv1.CollectorsGroupRoleNodeCollector {
				return odigosv1.CollectorsGroupRoleNodeCollector
			}
		}
	}
	return odigosv1.CollectorsGroupRoleClusterGateway
}

// NodeCollectorsGroupSpanMetricsEnabled reports whether the node CollectorsGroup has span metrics configured.
func NodeCollectorsGroupSpanMetricsEnabled(nodeCG *odigosv1.CollectorsGroup) bool {
	return nodeCG != nil && nodeCG.Spec.Metrics != nil && nodeCG.Spec.Metrics.SpanMetrics != nil
}
