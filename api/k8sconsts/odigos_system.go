package k8sconsts

const (
	OdigosSystemLabelKey       = "odigos.io/system-object"
	OdigosSystemConfigLabelKey = "odigos.io/config"
	OdigosSystemLabelValue     = "true"

	// New: mark resources that must not be pruned on upgrade
	OdigosPreserveLabelKey = "odigos.io/preserve"
)
