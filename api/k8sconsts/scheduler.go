package k8sconsts

const (
	SchedulerImage                  = "keyval/odigos-scheduler"
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
