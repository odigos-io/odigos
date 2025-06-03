package rollout

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
func Do(ctx context.Context, c client.Client, ic *odigosv1alpha1.InstrumentationConfig, pw k8sconsts.PodWorkload, rollbackDisabled bool) (bool, ctrl.Result, error) {
	logger := log.FromContext(ctx)
	workloadObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	err := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, workloadObj)
	if err != nil {
		return false, ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if ic == nil {
		// If ic is nil and the PodWorkload is missing the odigos.io/agents-meta-hash label,
		// it means it is a rolled back application that shouldn't be rolled out again.
		hasAgents, err := workloadHasOdigosAgents(ctx, c, workloadObj)
		if err != nil {
			logger.Error(err, "failed to check for odigos agent labels")
			return false, ctrl.Result{}, err
		}
		if !hasAgents {
			logger.Info("skipping rollout - workload already runs without odigos agents",
				"workload", pw.Name, "namespace", pw.Namespace)
			return false, ctrl.Result{}, nil
		}

		// instrumentation config is deleted, trigger a rollout for the associated workload
		// this should happen once per workload, as the instrumentation config is deleted
		// and we want to rollout the workload to remove the instrumentation
		rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
		return false, ctrl.Result{}, client.IgnoreNotFound(rolloutErr)
	}

	savedRolloutHash := ic.Status.WorkloadRolloutHash
	newRolloutHash := ic.Spec.AgentsMetaHash
	if savedRolloutHash == newRolloutHash {
		if !isWorkloadRolloutDone(workloadObj) && !rollbackDisabled {
			TimeSinceCrashLoopBackOff, err := crashLoopBackOffDuration(ctx, c, workloadObj)
			if err != nil {
				logger.Error(err, "Failed to check crashLoopBackOff")
				return false, ctrl.Result{}, err
			}

			if TimeSinceCrashLoopBackOff > 0 && ic.Spec.AgentInjectionEnabled {
				// Allow grace time for worklaod to stabelize before uninstrumenting it
				conf, err := k8sutils.GetCurrentOdigosConfig(ctx, c)
				if err != nil {
					return false, ctrl.Result{}, err
				}
				rollbackGraceTime := consts.AutoRollbackGraceTime
				if conf.RollbackGraceTime != "" {
					rollbackGraceTime = conf.RollbackGraceTime
				}
				graceTime, err := time.ParseDuration(rollbackGraceTime)
				if err != nil {
					return false, ctrl.Result{}, fmt.Errorf("invalid duration %q: %w", rollbackGraceTime, err)
				}

				if TimeSinceCrashLoopBackOff < graceTime {
					return false, ctrl.Result{RequeueAfter: graceTime - TimeSinceCrashLoopBackOff}, nil
				}

				for i := range ic.Spec.Containers {
					ic.Spec.Containers[i].AgentEnabled = false
					ic.Spec.Containers[i].AgentEnabledReason = odigosv1alpha1.AgentEnabledReasonCrashLoopBackOff
				}
				ic.Spec.AgentInjectionEnabled = false

				if err := c.Update(ctx, ic); err != nil {
					logger.Error(err, "failed to persist spec rollback")
					res, err := utils.K8SUpdateErrorHandler(err)
					return false, res, err
				}

				rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
				if rolloutErr != nil {
					logger.Error(rolloutErr, "error rolling out workload", "name", pw.Name, "namespace", pw.Namespace)
					return false, ctrl.Result{}, rolloutErr
				}

				meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
					Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
					Status:  metav1.ConditionTrue,
					Reason:  string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully),
					Message: "pods entered CrashLoopBackOff; instrumentation disabled",
				})

				meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
					Type:    odigosv1alpha1.AgentEnabledStatusConditionType,
					Status:  metav1.ConditionFalse,
					Reason:  string(odigosv1alpha1.AgentEnabledReasonCrashLoopBackOff),
					Message: "Pods entered CrashLoopBackOff; instrumentation disabled",
				})
				// Status always changes, requeue to test wait for status change with workload
				return true, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
			}
			// Requeue to wait for workload to finish or enter CrashLoopBackOff
			return false, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
		}
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
	// Requeue to try and catch a crashing app
	return true, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
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

// podHasCrashLoop returns true if any (init)-container in the pod is in CrashLoopBackOff.
func podHasCrashLoop(p *corev1.Pod) bool {
	for _, cs := range append(p.Status.InitContainerStatuses, p.Status.ContainerStatuses...) {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			return true
		}
	}
	return false
}

// crashLoopBackOffDuration returns how long the supplied workload
// (Deployment, StatefulSet, or DaemonSet) has been in *CrashLoopBackOff*.
//
// It inspects all Pods selected by the workload’s label selector:
//
//   - If at least one Pod is currently in CrashLoopBackOff, it finds the
//     earliest Pod.StartTime among those Pods and returns the elapsed time
//     since that moment.
//
//   - If **no** Pod is in CrashLoopBackOff, it simply returns 0 and no error.
//
// A non-nil error is returned only for unexpected situations (e.g. unsupported
// workload kind, invalid selector, or failed Pod list call).
func crashLoopBackOffDuration(ctx context.Context, c client.Client, obj client.Object) (time.Duration, error) {
	var (
		ns       string
		selector *metav1.LabelSelector
	)

	switch o := obj.(type) {
	case *appsv1.Deployment:
		ns, selector = o.Namespace, o.Spec.Selector
	case *appsv1.StatefulSet:
		ns, selector = o.Namespace, o.Spec.Selector
	case *appsv1.DaemonSet:
		ns, selector = o.Namespace, o.Spec.Selector
	default:
		return 0, fmt.Errorf("crashLoopBackOffDuration: unsupported workload kind %T", obj)
	}

	if selector == nil {
		return 0, fmt.Errorf("crashLoopBackOffDuration: workload has nil selector")
	}
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return 0, fmt.Errorf("crashLoopBackOffDuration: invalid selector: %w", err)
	}

	// 2. List matching Pods once (single API call for both checks).
	var podList corev1.PodList
	if err := c.List(ctx, &podList,
		client.InNamespace(ns),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		return 0, fmt.Errorf("crashLoopBackOffDuration: failed listing pods: %w", err)
	}

	// 3. Find the earliest-started Pod that is crashing.
	var earliest *time.Time
	for i := range podList.Items {
		p := &podList.Items[i]

		if !podHasCrashLoop(p) {
			continue
		}
		if p.Status.StartTime == nil { // extremely rare
			continue
		}

		start := p.Status.StartTime.Time
		if earliest == nil || start.Before(*earliest) {
			earliest = &start
		}
	}

	// 4. Return 0 if nothing is in CrashLoopBackOff.
	if earliest == nil {
		return 0, nil
	}

	// 5. Otherwise, duration since the workload entered CrashLoopBackOff.
	return time.Since(*earliest), nil
}

// workloadHasOdigosAgents returns true if the workload still has *any* pod present in the instrumented-pod.
func workloadHasOdigosAgents(ctx context.Context, c client.Client, obj client.Object) (bool, error) {
	var (
		ns       string
		selector *metav1.LabelSelector
	)

	switch o := obj.(type) {
	case *appsv1.Deployment:
		ns, selector = o.Namespace, o.Spec.Selector
	case *appsv1.StatefulSet:
		ns, selector = o.Namespace, o.Spec.Selector
	case *appsv1.DaemonSet:
		ns, selector = o.Namespace, o.Spec.Selector
	default:
		return false, fmt.Errorf("workloadHasOdigosAgents: unsupported workload kind %T", obj)
	}

	if selector == nil {
		return false, errors.New("workloadHasOdigosAgents: workload has nil selector")
	}

	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, fmt.Errorf("workloadHasOdigosAgents: invalid selector: %w", err)
	}

	var pods corev1.PodList
	if err := c.List(
		ctx, &pods,
		client.InNamespace(ns),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		return false, fmt.Errorf("workloadHasOdigosAgents: listing pods failed: %w", err)
	}

	// Because the cache already filters on k8sconsts.OdigosAgentsMetaHashLabel,
	// any non-empty list means the workload still runs instrumented pods.
	return len(pods.Items) > 0, nil
}
