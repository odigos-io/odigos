package workloadrollout

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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