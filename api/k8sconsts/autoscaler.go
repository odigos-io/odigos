package k8sconsts

const (
	AutoScalerDeploymentName              = "odigos-autoscaler"
	AutoScalerImageCertified              = "odigos-autoscaler-certified"
	AutoScalerImageName                   = "odigos-autoscaler"
	AutoScalerServiceAccountName          = AutoScalerDeploymentName
	AutoScalerAppLabelValue               = AutoScalerDeploymentName
	AutoScalerRoleName                    = AutoScalerDeploymentName
	AutoScalerRoleBindingName             = AutoScalerDeploymentName
	AutoScalerClusterRoleName             = AutoScalerDeploymentName
	AutoScalerClusterRoleBindingName      = AutoScalerDeploymentName
	AutoscalerCAName                      = AutoScalerDeploymentName
	AutoScalerServiceName                 = "auto-scaler"
	AutoScalerContainerName               = "manager"
	AutoscalerActionValidatingWebhookName = "odigos-action-validating-webhook-configuration"

	// The webhooks certificate secret was renamed mainly to migrate away from
	// having the secret as a helm hook.
	// Deprecated: only use for migration purposes.
	DeprecatedAutoscalerWebhookSecretName = "autoscaler-webhook-cert"
	DeprecatedAutoscalerWebhookVolumeName = "autoscaler-webhook-cert"
	AutoscalerWebhookSecretName           = "autoscaler-webhooks-cert"
	AutoscalerWebhookVolumeName           = "autoscaler-webhooks-cert"
	AutoScalerWebhookServiceName          = "odigos-autoscaler"
	AutoScalerWebhookFieldOwner           = AutoScalerDeploymentName
)

const (
	K8sAttributesFromDefaultValue = "pod"
)
