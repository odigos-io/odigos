package k8sconsts

const (
	InstrumentorOtelServiceName             = "instrumentor"
	InstrumentorDeploymentName              = "odigos-instrumentor"
	InstrumentorImage                       = "keyval/odigos-instrumentor"
	InstrumentorEnterpriseImage             = "keyval/odigos-enterprise-instrumentor"
	InstrumentorImageUBI9                   = "odigos-instrumentor-ubi9"
	InstrumentorEnterpriseImageUBI9         = "odigos-enterprise-instrumentor-ubi9"
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
