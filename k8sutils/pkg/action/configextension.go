package action

import (
	actionscatalog "github.com/odigos-io/odigos/actions"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	odigoscommon "github.com/odigos-io/odigos/common"
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

// SupportedSignals returns the signals the action catalog declares this Action's type can
// process. The Action CRD accepts any signal, but a collector processor that cannot consume a
// signal makes the collector fail to build that pipeline ("telemetry type is not supported") and
// crash-loop, so callers must intersect the requested signals with these before wiring a
// processor into the collector config.
func SupportedSignals(action *odigosv1.Action) []odigoscommon.ObservabilitySignal {
	manifest, ok := actionManifestForCRD(action)
	if !ok {
		return nil
	}

	signals := make([]odigoscommon.ObservabilitySignal, 0, 4)
	if manifest.Spec.Signals.Traces.Supported {
		signals = append(signals, odigoscommon.TracesObservabilitySignal)
	}
	if manifest.Spec.Signals.Metrics.Supported {
		signals = append(signals, odigoscommon.MetricsObservabilitySignal)
	}
	if manifest.Spec.Signals.Logs.Supported {
		signals = append(signals, odigoscommon.LogsObservabilitySignal)
	}
	if manifest.Spec.Signals.Profiles.Supported {
		signals = append(signals, odigoscommon.ProfilesObservabilitySignal)
	}
	return signals
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
