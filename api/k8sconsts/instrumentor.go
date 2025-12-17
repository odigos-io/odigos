package k8sconsts

const (
	InstrumentorOtelServiceName             = "instrumentor"
	InstrumentorDeploymentName              = "odigos-instrumentor"
	InstrumentorImage                       = "odigos-instrumentor"
	InstrumentorEnterpriseImage             = "odigos-enterprise-instrumentor"
	InstrumentorImageCertified              = "odigos-instrumentor-certified"
	InstrumentorEnterpriseImageCertified    = "odigos-enterprise-instrumentor-certified"
	InstrumentorAppLabelValue               = InstrumentorDeploymentName
	InstrumentorServiceName                 = InstrumentorDeploymentName
	InstrumentorServiceAccountName          = InstrumentorDeploymentName
	InstrumentorRoleName                    = InstrumentorDeploymentName
	InstrumentorRoleBindingName             = InstrumentorDeploymentName
	InstrumentorClusterRoleName             = InstrumentorDeploymentName
	InstrumentorClusterRoleBindingName      = InstrumentorDeploymentName
	InstrumentorCAName                      = InstrumentorDeploymentName
	InstrumentorWebhookFieldOwner           = InstrumentorDeploymentName
	InstrumentorMutatingWebhookName         = "odigos-pod-mutating-webhook-configuration"
	InstrumentorSourceMutatingWebhookName   = "odigos-source-mutating-webhook-configuration"
	InstrumentorSourceValidatingWebhookName = "odigos-source-validating-webhook-configuration"
	InstrumentorContainerName               = "manager"

	InstrumentorWebhookSecretName = "instrumentor-webhooks-cert"
	InstrumentorWebhookVolumeName = "instrumentor-webhooks-cert"

	// Deprecated: only use for migration purposes.
	DeprecatedInstrumentorWebhookSecretName = "webhook-cert"
	DeprecatedInstrumentorWebhookVolumeName = "webhook-cert"
)
