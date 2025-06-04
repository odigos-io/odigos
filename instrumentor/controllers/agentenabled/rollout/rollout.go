package rollout

import (
	"context"
	"errors"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

const requeueWaitingForWorkloadRollout = 10 * time.Second

// Do potentially triggers a rollout for the given workload based on the given instrumentation config.
// If the instrumentation config is nil, the workload is rolled out - this is used for un-instrumenting workloads.
// Otherwise, the rollout config hash is calculated and compared to the saved hash in the instrumentation config.
// If the hashes are different, the workload is rolled out.
// If the hashes are the same, this is a no-op.
//
// If a rollout is triggered the status of the instrumentation config is updated with the new rollout hash
// and a corresponding condition is set.
//
// Returns a boolean indicating if the status of the instrumentation config has changed, a ctrl.Result and an error.
func Do(ctx context.Context, c client.Client, ic *odigosv1alpha1.InstrumentationConfig, pw k8sconsts.PodWorkload) (bool, ctrl.Result, error) {
	logger := log.FromContext(ctx)
	workloadObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	err := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, workloadObj)
	if err != nil {
		return false, ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if ic == nil {
		// instrumentation config is deleted, trigger a rollout for the associated workload
		// this should happen once per workload, as the instrumentation config is deleted
		// and we want to rollout the workload to remove the instrumentation
		rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
		return false, ctrl.Result{}, client.IgnoreNotFound(rolloutErr)
	}

	savedRolloutHash := ic.Status.WorkloadRolloutHash
	newRolloutHash := ic.Spec.AgentsMetaHash

	if savedRolloutHash == newRolloutHash {
		return false, ctrl.Result{}, nil
	}

	// if a rollout is ongoing, wait for it to finish, requeue
	statusChanged := false
	if !isWorkloadRolloutDone(workloadObj) {
		statusChanged = meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
			Status:  metav1.ConditionUnknown,
			Reason:  string(odigosv1alpha1.WorkloadRolloutReasonPreviousRolloutOngoing),
			Message: "waiting for workload rollout to finish before triggering a new one",
		})
		return statusChanged, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
	}

	rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
	if rolloutErr != nil {
		logger.Error(rolloutErr, "error rolling out workload", "name", pw.Name, "namespace", pw.Namespace)
	}

	ic.Status.WorkloadRolloutHash = newRolloutHash
	meta.SetStatusCondition(&ic.Status.Conditions, rolloutCondition(rolloutErr))

	// at this point, the hashes are different, notify the caller the status has changed
	return true, ctrl.Result{}, nil
}

// RolloutRestartWorkload restarts the given workload by patching its template annotations.
// this is bases on the kubectl implementation of restarting a workload
// https://github.com/kubernetes/kubectl/blob/master/pkg/polymorphichelpers/objectrestarter.go#L32
func rolloutRestartWorkload(ctx context.Context, workload client.Object, c client.Client, ts time.Time) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, ts.Format(time.RFC3339)))
	switch obj := workload.(type) {
	case *appsv1.Deployment:
		if obj.Spec.Paused {
			return errors.New("can't restart paused deployment")
		}
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
	case *appsv1.StatefulSet:
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
	case *appsv1.DaemonSet:
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
	default:
		return errors.New("unknown kind")
	}
}

// isWorkloadRolloutDone checks if the rollout of the given workload is done.
// this is based on the kubectl implementation of checking the rollout status:
// https://github.com/kubernetes/kubectl/blob/master/pkg/polymorphichelpers/rollout_status.go
func isWorkloadRolloutDone(obj client.Object) bool {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		if o.Generation <= o.Status.ObservedGeneration {
			cond := conditions.GetDeploymentCondition(o.Status, appsv1.DeploymentProgressing)
			if cond != nil && cond.Reason == conditions.TimedOutReason {
				// deployment exceeded its progress deadline
				return false
			}
			if o.Spec.Replicas != nil && o.Status.UpdatedReplicas < *o.Spec.Replicas {
				// Waiting for deployment rollout to finish
				return false
			}
			if o.Status.Replicas > o.Status.UpdatedReplicas {
				// Waiting for deployment rollout to finish old replicas are pending termination.
				return false
			}
			if o.Status.AvailableReplicas < o.Status.UpdatedReplicas {
				// Waiting for deployment rollout to finish:  not all updated replicas are available..
				return false
			}
			return true
		}
		return false
	case *appsv1.StatefulSet:
		if o.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
			// rollout status is only available for RollingUpdateStatefulSetStrategyType strategy type
			return true
		}
		if o.Status.ObservedGeneration == 0 || o.Generation > o.Status.ObservedGeneration {
			// Waiting for statefulset spec update to be observed
			return false
		}
		if o.Spec.Replicas != nil && o.Status.ReadyReplicas < *o.Spec.Replicas {
			// Waiting for pods to be ready
			return false
		}
		if o.Spec.UpdateStrategy.Type == appsv1.RollingUpdateStatefulSetStrategyType && o.Spec.UpdateStrategy.RollingUpdate != nil {
			if o.Spec.Replicas != nil && o.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
				if o.Status.UpdatedReplicas < (*o.Spec.Replicas - *o.Spec.UpdateStrategy.RollingUpdate.Partition) {
					// Waiting for partitioned roll out to finish
					return false
				}
			}
			// partitioned roll out complete
			return true
		}
		if o.Status.UpdateRevision != o.Status.CurrentRevision {
			// waiting for statefulset rolling update to complete
			return false
		}
		return true
	case *appsv1.DaemonSet:
		if o.Spec.UpdateStrategy.Type != appsv1.RollingUpdateDaemonSetStrategyType {
			// rollout status is only available for RollingUpdateDaemonSetStrategyType strategy type
			return true
		}
		if o.Generation <= o.Status.ObservedGeneration {
			if o.Status.UpdatedNumberScheduled < o.Status.DesiredNumberScheduled {
				// Waiting for daemon set rollout to finish
				return false
			}
			if o.Status.NumberAvailable < o.Status.DesiredNumberScheduled {
				// Waiting for daemon set rollout to finish
				return false
			}
			return true
		}
		// Waiting for daemon set spec update to be observed
		return false
	default:
		return false
	}
}

func rolloutCondition(rolloutErr error) metav1.Condition {
	cond := metav1.Condition{
		Type: odigosv1alpha1.WorkloadRolloutStatusConditionType,
	}

	if rolloutErr == nil {
		cond.Status = metav1.ConditionTrue
		cond.Reason = string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully)
		cond.Message = "workload rollout triggered successfully"
	} else {
		cond.Status = metav1.ConditionFalse
		cond.Reason = string(odigosv1alpha1.WorkloadRolloutReasonFailedToPatch)
		cond.Message = rolloutErr.Error()
	}

	return cond
}
