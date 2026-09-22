package k8sconsts

const (
	OdigosSystemLabelKey       = "odigos.io/system-object"
	OdigosSystemConfigLabelKey = "odigos.io/config"
	OdigosSystemLabelValue     = "true"

	// OdigosCopiedImagePullSecretLabel marks pull secrets Odigos copied into
	// instrumented namespaces. Those copies may be updated on re-sync; secrets
	// with the same name but without this label are left alone.
	OdigosCopiedImagePullSecretLabel = "odigos.io/copied-image-pull-secret"
)
