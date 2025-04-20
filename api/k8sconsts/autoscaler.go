package k8sconsts

const (
	AutoScalerDeploymentName         = "odigos-autoscaler"
	AutoScalerImageUBI9              = "odigos-autoscaler-ubi9"
	AutoScalerImageName              = "odigos-autoscaler"
	AutoScalerServiceAccountName     = AutoScalerDeploymentName
	AutoScalerAppLabelValue          = AutoScalerDeploymentName
	AutoScalerRoleName               = AutoScalerDeploymentName
	AutoScalerRoleBindingName        = AutoScalerDeploymentName
	AutoScalerClusterRoleName        = AutoScalerDeploymentName
	AutoScalerClusterRoleBindingName = AutoScalerDeploymentName
	AutoScalerServiceName            = "auto-scaler"
	AutoScalerContainerName          = "manager"

	DefaultK8sNodeLogsDirectory = "/var/log"
)
