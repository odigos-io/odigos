package k8sconsts

const (
	InstrumentorOtelServiceName             = "instrumentor"
	InstrumentorDeploymentName              = "odigos-instrumentor"
	InstrumentorImage                       = "odigos-instrumentor"
	InstrumentorEnterpriseImage             = "odigos-enterprise-instrumentor"
	InstrumentorImageUBI9                   = "odigos-instrumentor-ubi9"
	InstrumentorEnterpriseImageUBI9         = "odigos-enterprise-instrumentor-ubi9"
	InstrumentorAppLabelValue               = InstrumentorDeploymentName
	InstrumentorServiceName                 = InstrumentorDeploymentName
	InstrumentorServiceAccountName          = InstrumentorDeploymentName
	InstrumentorRoleName                    = InstrumentorDeploymentName
	InstrumentorRoleBindingName             = InstrumentorDeploymentName
	InstrumentorClusterRoleName             = InstrumentorDeploymentName
	InstrumentorClusterRoleBindingName      = InstrumentorDeploymentName
	InstrumentorCAName                      = InstrumentorDeploymentName
	InstrumentorWebhookFieldOwner           = InstrumentorDeploymentName
	InstrumentorMutatingWebhookName         = "mutating-webhook-configuration"
	InstrumentorSourceMutatingWebhookName   = "source-mutating-webhook-configuration"
	InstrumentorSourceValidatingWebhookName = "source-validating-webhook-configuration"
	InstrumentorContainerName               = "manager"

	InstrumentorWebhookSecretName = "instrumentor-webhooks-cert"
	InstrumentorWebhookVolumeName = "instrumentor-webhooks-cert"

	// Deprecated: only use for migration purposes.
	DeprecatedInstrumentorWebhookSecretName = "webhook-cert"
	DeprecatedInstrumentorWebhookVolumeName = "webhook-cert"
)
