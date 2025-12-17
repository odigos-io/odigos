package k8sconsts

const (
	SchedulerImage                  = "odigos-scheduler"
	SchedulerImageCertified         = "odigos-scheduler-certified"
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
