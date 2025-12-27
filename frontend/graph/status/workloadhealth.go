package status

import (
	"fmt"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	WorkloadHealthStatus = "WorkloadHealth"
)

type WorkloadHealthStatusReason string

const (
	WorkloadHealthStatusReasonHealthy                     WorkloadHealthStatusReason = "Healthy"
	WorkloadHealthStatusReasonNoAvailableReplicas         WorkloadHealthStatusReason = "NoAvailableReplicas"
	WorkloadHealthStatusReasonProgressing                 WorkloadHealthStatusReason = "Progressing"
	WorkloadHealthStatusReasonProgressingDeadlineExceeded WorkloadHealthStatusReason = "ProgressingDeadlineExceeded"
	WorkloadHealthStatusReasonReplicaFailure              WorkloadHealthStatusReason = "ReplicaFailure"
	WorkloadHealthStatusReasonProgressingError            WorkloadHealthStatusReason = "ProgressingError"
)

func CalculateDeploymentHealthStatus(deploymentStatus appsv1.DeploymentStatus) *model.DesiredConditionStatus {

	for _, condition := range deploymentStatus.Conditions {

		switch condition.Type {
		case appsv1.DeploymentAvailable:
			if condition.Status == corev1.ConditionFalse {
				reasonStr := string(WorkloadHealthStatusReasonNoAvailableReplicas)
				return &model.DesiredConditionStatus{
					Name:       WorkloadHealthStatus,
					ReasonEnum: &reasonStr,
					Status:     model.DesiredStateProgressFailure,
					Message:    "Deployment does not have at least the minimum number of available replicas required",
				}
			}

		case appsv1.DeploymentProgressing:
			switch condition.Reason {
			case "ProgressDeadlineExceeded":
				reasonStr := string(WorkloadHealthStatusReasonProgressingDeadlineExceeded)
				return &model.DesiredConditionStatus{
					Name:       WorkloadHealthStatus,
					ReasonEnum: &reasonStr,
					Status:     model.DesiredStateProgressFailure,
					Message:    "Deployment failed to start new pods after the deadline",
				}
			case "ReplicaSetUpdated":
				reasonStr := string(WorkloadHealthStatusReasonProgressing)
				return &model.DesiredConditionStatus{
					Name:       WorkloadHealthStatus,
					ReasonEnum: &reasonStr,
					Status:     model.DesiredStateProgressWaiting,
					Message:    "Deployment is progressing after rollout or revision update, new pods are not yet available",
				}
			}
			if condition.Status != corev1.ConditionTrue {
				reasonStr := string(WorkloadHealthStatusReasonProgressingError)
				return &model.DesiredConditionStatus{
					Name:       WorkloadHealthStatus,
					ReasonEnum: &reasonStr,
					Status:     model.DesiredStateProgressFailure,
					Message:    "Deployment progressing is unhealthy",
				}
			}

		case appsv1.DeploymentReplicaFailure:
			reasonStr := string(WorkloadHealthStatusReasonReplicaFailure)
			return &model.DesiredConditionStatus{
				Name:       WorkloadHealthStatus,
				ReasonEnum: &reasonStr,
				Status:     model.DesiredStateProgressFailure,
				Message:    "Deployment has pods which failed to be created or deleted",
			}
		}
	}

	// check if all the replicas are ok, or if there are some errors / progress in replicas count
	if deploymentStatus.UnavailableReplicas > 0 {
		reasonStr := string(WorkloadHealthStatusReasonNoAvailableReplicas)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("Deployment has %d/%d unavailable replicas", deploymentStatus.UnavailableReplicas, deploymentStatus.Replicas),
		}
	}

	if deploymentStatus.Replicas != deploymentStatus.UpdatedReplicas ||
		deploymentStatus.Replicas != deploymentStatus.ReadyReplicas ||
		deploymentStatus.Replicas != deploymentStatus.AvailableReplicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    "not all deployment replicase are avaiable and ready",
		}
	}

	reasonStr := string(WorkloadHealthStatusReasonHealthy)
	return &model.DesiredConditionStatus{
		Name:       WorkloadHealthStatus,
		ReasonEnum: &reasonStr,
		Status:     model.DesiredStateProgressSuccess,
		Message:    "All deployment replicase are available and ready",
	}
}

func CalculateDaemonSetHealthStatus(daemonSetStatus appsv1.DaemonSetStatus) *model.DesiredConditionStatus {

	// check if all the replicas are ok, or if there are some errors / progress in replicas count
	if daemonSetStatus.DesiredNumberScheduled == 0 {
		reasonStr := string(WorkloadHealthStatusReasonNoAvailableReplicas)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressIrrelevant,
			Message:    "DaemonSet has no desired replicas scheduled",
		}
	}

	// Check if current scheduled replicas match desired
	if daemonSetStatus.CurrentNumberScheduled < daemonSetStatus.DesiredNumberScheduled {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("DaemonSet has %d/%d current replicas scheduled", daemonSetStatus.CurrentNumberScheduled, daemonSetStatus.DesiredNumberScheduled),
		}
	}

	// Check if updated replicas match desired (for rolling updates)
	if daemonSetStatus.UpdatedNumberScheduled < daemonSetStatus.DesiredNumberScheduled {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("DaemonSet has %d/%d updated replicas", daemonSetStatus.UpdatedNumberScheduled, daemonSetStatus.DesiredNumberScheduled),
		}
	}

	// Check if available replicas match desired
	if daemonSetStatus.NumberAvailable < daemonSetStatus.DesiredNumberScheduled {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("DaemonSet has %d/%d available replicas", daemonSetStatus.NumberAvailable, daemonSetStatus.DesiredNumberScheduled),
		}
	}

	// Check if ready replicas match desired
	if daemonSetStatus.NumberReady < daemonSetStatus.DesiredNumberScheduled {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("DaemonSet has %d/%d ready replicas", daemonSetStatus.NumberReady, daemonSetStatus.DesiredNumberScheduled),
		}
	}

	reasonStr := string(WorkloadHealthStatusReasonHealthy)
	return &model.DesiredConditionStatus{
		Name:       WorkloadHealthStatus,
		ReasonEnum: &reasonStr,
		Status:     model.DesiredStateProgressSuccess,
		Message:    "DaemonSet replicas are reported healthy in kubernetes",
	}
}

func CalculateStatefulSetHealthStatus(statefulSetStatus appsv1.StatefulSetStatus) *model.DesiredConditionStatus {

	// Check if there are any replicas at all
	if statefulSetStatus.Replicas == 0 {
		reasonStr := string(WorkloadHealthStatusReasonNoAvailableReplicas)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressIrrelevant,
			Message:    "StatefulSet has no replicas",
		}
	}

	// Check if ready replicas match total replicas
	if statefulSetStatus.ReadyReplicas < statefulSetStatus.Replicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("StatefulSet has %d/%d ready replicas", statefulSetStatus.ReadyReplicas, statefulSetStatus.Replicas),
		}
	}

	// Check if available replicas match total replicas
	if statefulSetStatus.AvailableReplicas < statefulSetStatus.Replicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("StatefulSet has %d/%d available replicas", statefulSetStatus.AvailableReplicas, statefulSetStatus.Replicas),
		}
	}

	// Check if updated replicas match total replicas (for rolling updates)
	if statefulSetStatus.UpdatedReplicas < statefulSetStatus.Replicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("StatefulSet has %d/%d updated replicas", statefulSetStatus.UpdatedReplicas, statefulSetStatus.Replicas),
		}
	}

	// Check if current and update revisions match (for rolling updates)
	if statefulSetStatus.CurrentRevision != statefulSetStatus.UpdateRevision {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    "StatefulSet is in the middle of a rolling update",
		}
	}

	reasonStr := string(WorkloadHealthStatusReasonHealthy)
	return &model.DesiredConditionStatus{
		Name:       WorkloadHealthStatus,
		ReasonEnum: &reasonStr,
		Status:     model.DesiredStateProgressSuccess,
		Message:    "StatefulSet replicas are reported healthy in kubernetes",
	}
}

func CalculateCronJobHealthStatus(cronJobStatus batchv1.CronJobStatus) *model.DesiredConditionStatus {

	// Check if there are any active jobs
	if len(cronJobStatus.Active) > 0 {
		reasonStr := string(WorkloadHealthStatusReasonHealthy)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressSuccess,
			Message:    fmt.Sprintf("CronJob has %d active jobs running", len(cronJobStatus.Active)),
		}
	}

	// Check if the CronJob has ever been scheduled
	if cronJobStatus.LastScheduleTime == nil {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressPending,
			Message:    "CronJob has never been scheduled",
		}
	}

	// Check if there's a last successful time (indicates the CronJob has run successfully at least once)
	if cronJobStatus.LastSuccessfulTime != nil {
		reasonStr := string(WorkloadHealthStatusReasonHealthy)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressSuccess,
			Message:    "CronJob is healthy and has completed successfully",
		}
	}

	// If we have a last schedule time but no successful time, it might be waiting for the first run
	reasonStr := string(WorkloadHealthStatusReasonProgressing)
	return &model.DesiredConditionStatus{
		Name:       WorkloadHealthStatus,
		ReasonEnum: &reasonStr,
		Status:     model.DesiredStateProgressPending,
		Message:    "CronJob is waiting for next scheduled run",
	}
}

func CalculateDeploymentConfigHealthStatus(dcStatus openshiftappsv1.DeploymentConfigStatus) *model.DesiredConditionStatus {

	// Check for available condition
	for _, condition := range dcStatus.Conditions {
		switch condition.Type {
		case openshiftappsv1.DeploymentAvailable:
			if condition.Status == corev1.ConditionFalse {
				reasonStr := string(WorkloadHealthStatusReasonNoAvailableReplicas)
				return &model.DesiredConditionStatus{
					Name:       WorkloadHealthStatus,
					ReasonEnum: &reasonStr,
					Status:     model.DesiredStateProgressFailure,
					Message:    "DeploymentConfig does not have at least the minimum number of available replicas required",
				}
			}

		case openshiftappsv1.DeploymentProgressing:
			if condition.Status != corev1.ConditionTrue {
				reasonStr := string(WorkloadHealthStatusReasonProgressingError)
				return &model.DesiredConditionStatus{
					Name:       WorkloadHealthStatus,
					ReasonEnum: &reasonStr,
					Status:     model.DesiredStateProgressFailure,
					Message:    "DeploymentConfig progressing is unhealthy",
				}
			}

		case openshiftappsv1.DeploymentReplicaFailure:
			reasonStr := string(WorkloadHealthStatusReasonReplicaFailure)
			return &model.DesiredConditionStatus{
				Name:       WorkloadHealthStatus,
				ReasonEnum: &reasonStr,
				Status:     model.DesiredStateProgressFailure,
				Message:    "DeploymentConfig has pods which failed to be created or deleted",
			}
		}
	}

	// check if all the replicas are ok, or if there are some errors / progress in replicas count
	if dcStatus.UnavailableReplicas > 0 {
		reasonStr := string(WorkloadHealthStatusReasonNoAvailableReplicas)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("DeploymentConfig has %d/%d unavailable replicas", dcStatus.UnavailableReplicas, dcStatus.Replicas),
		}
	}

	if dcStatus.Replicas != dcStatus.UpdatedReplicas ||
		dcStatus.Replicas != dcStatus.ReadyReplicas ||
		dcStatus.Replicas != dcStatus.AvailableReplicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    "not all deploymentconfig replicas are available and ready",
		}
	}

	reasonStr := string(WorkloadHealthStatusReasonHealthy)
	return &model.DesiredConditionStatus{
		Name:       WorkloadHealthStatus,
		ReasonEnum: &reasonStr,
		Status:     model.DesiredStateProgressSuccess,
		Message:    "All deploymentconfig replicas are available and ready",
	}
}

func CalculateRolloutHealthStatus(rolloutStatus argorolloutsv1alpha1.RolloutStatus) *model.DesiredConditionStatus {

	if rolloutStatus.Replicas == 0 {
		reasonStr := string(WorkloadHealthStatusReasonNoAvailableReplicas)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressIrrelevant,
			Message:    "Argo Rollout has no replicas",
		}
	}

	// Check if available replicas match total replicas
	if rolloutStatus.AvailableReplicas < rolloutStatus.Replicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("Argo Rollout has %d/%d available replicas", rolloutStatus.AvailableReplicas, rolloutStatus.Replicas),
		}
	}

	// Check if updated replicas match total replicas (for rolling updates)
	if rolloutStatus.UpdatedReplicas < rolloutStatus.Replicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("Argo Rollout has %d/%d updated replicas", rolloutStatus.UpdatedReplicas, rolloutStatus.Replicas),
		}
	}

	// Check if ready replicas match total replicas
	if rolloutStatus.ReadyReplicas < rolloutStatus.Replicas {
		reasonStr := string(WorkloadHealthStatusReasonProgressing)
		return &model.DesiredConditionStatus{
			Name:       WorkloadHealthStatus,
			ReasonEnum: &reasonStr,
			Status:     model.DesiredStateProgressWaiting,
			Message:    fmt.Sprintf("Rollout has %d/%d ready replicas", rolloutStatus.ReadyReplicas, rolloutStatus.Replicas),
		}
	}

	reasonStr := string(WorkloadHealthStatusReasonHealthy)
	return &model.DesiredConditionStatus{
		Name:       WorkloadHealthStatus,
		ReasonEnum: &reasonStr,
		Status:     model.DesiredStateProgressSuccess,
		Message:    "All Argo Rollout replicas are available and ready",
	}
}
