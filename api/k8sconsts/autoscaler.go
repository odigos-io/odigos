package k8sconsts

const (
	AutoScalerDeploymentName              = "odigos-autoscaler"
	AutoScalerImageUBI9                   = "odigos-autoscaler-ubi9"
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
	AutoscalerActionValidatingWebhookName = "action-validating-webhook-configuration"

	// The webhooks certificate secret was renamed mainly to migrate away from
	// having the secret as a helm hook.
	// Deprecated: only use for migration purposes.
	DeprecatedAutoscalerWebhookSecretName = "autoscaler-webhook-cert"
	DeprecatedAutoscalerWebhookVolumeName = "autoscaler-webhook-cert"
	AutoscalerWebhookSecretName		      = "autoscaler-webhooks-cert"
	AutoscalerWebhookVolumeName		      = "autoscaler-webhooks-cert"
	AutoScalerWebhookServiceName          = "odigos-autoscaler"
	AutoScalerWebhookFieldOwner           = AutoScalerDeploymentName
)
