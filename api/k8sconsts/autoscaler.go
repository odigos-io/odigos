package k8sconsts

const (
	AutoScalerDeploymentName         = "odigos-autoscaler"
	AutoScalerServiceAccountName     = AutoScalerDeploymentName
	AutoScalerAppLabelValue          = AutoScalerDeploymentName
	AutoScalerRoleName               = AutoScalerDeploymentName
	AutoScalerRoleBindingName        = AutoScalerDeploymentName
	AutoScalerClusterRoleName        = AutoScalerDeploymentName
	AutoScalerClusterRoleBindingName = AutoScalerDeploymentName
	AutoScalerServiceName            = "auto-scaler"
	AutoScalerContainerName          = "manager"
)
