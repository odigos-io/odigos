package k8sconsts

const (
	InstrumentorOtelServiceName             = "instrumentor"
	InstrumentorDeploymentName              = "odigos-instrumentor"
	InstrumentorAppLabelValue               = InstrumentorDeploymentName
	InstrumentorServiceName                 = InstrumentorDeploymentName
	InstrumentorServiceAccountName          = InstrumentorDeploymentName
	InstrumentorRoleName                    = InstrumentorDeploymentName
	InstrumentorRoleBindingName             = InstrumentorDeploymentName
	InstrumentorClusterRoleName             = InstrumentorDeploymentName
	InstrumentorClusterRoleBindingName      = InstrumentorDeploymentName
	InstrumentorCertificateName             = InstrumentorDeploymentName
	InstrumentorMutatingWebhookName         = "mutating-webhook-configuration"
	InstrumentorSourceMutatingWebhookName   = "source-mutating-webhook-configuration"
	InstrumentorSourceValidatingWebhookName = "source-validating-webhook-configuration"
	InstrumentorContainerName               = "manager"
	InstrumentorWebhookSecretName           = "webhook-cert"
	InstrumentorWebhookVolumeName           = "webhook-cert"
)
