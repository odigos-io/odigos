package k8sconsts

const (
	SchedulerImage                  = "registry.odigos.io/odigos-scheduler"
	SchedulerImageUBI9              = "odigos-scheduler-ubi9"
	SchedulerServiceName            = "scheduler"
	SchedulerDeploymentName         = "odigos-scheduler"
	SchedulerAppLabelValue          = SchedulerDeploymentName
	SchedulerRoleName               = SchedulerDeploymentName
	SchedulerRoleBindingName        = SchedulerDeploymentName
	SchedulerClusterRoleName        = SchedulerDeploymentName
	SchedulerClusterRoleBindingName = SchedulerDeploymentName
	SchedulerServiceAccountName     = SchedulerDeploymentName
	SchedulerContainerName          = "manager"
)
