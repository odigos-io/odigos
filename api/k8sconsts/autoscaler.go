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
	AutoscalerCertificateName             = AutoScalerDeploymentName
	AutoScalerServiceName                 = "auto-scaler"
	AutoScalerContainerName               = "manager"
	AutoscalerActionValidatingWebhookName = "action-validating-webhook-configuration"
	AutoscalerWebhookSecretName           = "autoscaler-webhook-cert"
	AutoscalerWebhookVolumeName           = "autoscaler-webhook-cert"
	AutoScalerWebhookServiceName          = "odigos-autoscaler"
)
