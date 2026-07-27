package action

import (
	actionscatalog "github.com/odigos-io/odigos/actions"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
)

const odigosConfigExtensionMechanism = "odigosConfigExtension"

// IsConfigExtension reports whether the action type uses the
// odigosConfigExtension config mechanism, regardless of whether it is disabled.
func IsConfigExtension(action *odigosv1.Action) bool {
	return len(ConfigExtensionProcessorTypes(action)) > 0
}

// ConfigExtensionProcessorTypes returns processor types from the action catalog
// that use the odigosConfigExtension config mechanism for this Action CR.
func ConfigExtensionProcessorTypes(action *odigosv1.Action) []string {
	manifest, ok := actionManifestForCRD(action)
	if !ok {
		return nil
	}

	processors := []string{}
	for _, processorMetadata := range manifest.Spec.Processors {
		if processorMetadata.ConfigMechanism == odigosConfigExtensionMechanism {
			processors = append(processors, processorMetadata.Type)
		}
	}
	return processors
}

func actionManifestForCRD(action *odigosv1.Action) (actionscatalog.Action, bool) {
	return actionscatalog.GetActionByType(CatalogType(action))
}

// CatalogType returns the catalog metadata.type for the configured Action config.
func CatalogType(action *odigosv1.Action) string {
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
