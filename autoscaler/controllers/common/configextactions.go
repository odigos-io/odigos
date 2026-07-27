package common

import (
	"encoding/json"
	"slices"

	"github.com/odigos-io/odigos/actions"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
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

// IsConfigExtension reports whether the action type uses the
// odigosConfigExtension config mechanism, regardless of whether it is disabled.
func IsConfigExtension(action *odigosv1.Action) bool {
	return len(configExtensionProcessorTypes(action)) > 0
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
	return configExtensionProcessorTypes(action)
}

func configExtensionProcessorTypes(action *odigosv1.Action) []string {
	manifest, ok := actionManifestForCRD(action)
	if !ok {
		return []string{}
	}

	processors := []string{}
	for _, processorMetadata := range manifest.Spec.Processors {
		if processorMetadata.ConfigMechanism == "odigosConfigExtension" {
			processors = append(processors, processorMetadata.Type)
		}
	}
	return processors
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

// actionManifestForCRD returns the static YAML catalog entry for the Action CRD's type.
func actionManifestForCRD(action *odigosv1.Action) (actions.Action, bool) {
	return actions.GetActionByType(actionCatalogType(action))
}

// actionCatalogType returns the catalog metadata.type for the configured Action config.
func actionCatalogType(action *odigosv1.Action) string {
	switch {
	case action.Spec.AddClusterInfo != nil:
		return actionsv1.ActionNameAddClusterInfo
	case action.Spec.DeleteAttribute != nil:
		return actionsv1.ActionNameDeleteAttribute
	case action.Spec.RenameAttribute != nil:
		return actionsv1.ActionNameRenameAttribute
	case action.Spec.PiiMasking != nil:
		return odigosactions.ActionNamePiiMasking
	case action.Spec.K8sAttributes != nil:
		return "K8sAttributesResolver"
	case action.Spec.URLTemplatization != nil:
		return odigosactions.ActionNameURLTemplatization
	case action.Spec.SpanRenamer != nil:
		return odigosactions.ActionSpanRenamer
	case action.Spec.ExtractAttribute != nil:
		return odigosactions.ActionNameExtractAttribute
	case action.Spec.DbQueryTemplatization != nil:
		return odigosactions.ActionNameDbQueryTemplatization
	case action.Spec.InferDbAttributes != nil:
		return odigosactions.ActionNameInferDbAttributes
	default:
		return ""
	}
}
