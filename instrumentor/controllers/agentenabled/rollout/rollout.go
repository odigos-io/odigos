package rollout

import (
	"context"
	"errors"
	"fmt"
	"time"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	containerutils "github.com/odigos-io/odigos/k8sutils/pkg/container"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
func Do(ctx context.Context, c client.Client, ic *odigosv1alpha1.InstrumentationConfig, pw k8sconsts.PodWorkload, conf *common.OdigosConfiguration, distroProvider *distros.Provider) (bool, ctrl.Result, error) {
	logger := log.FromContext(ctx)
	automaticRolloutDisabled := conf.Rollout != nil && conf.Rollout.AutomaticRolloutDisabled != nil && *conf.Rollout.AutomaticRolloutDisabled
	workloadObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	err := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, workloadObj)
	if err != nil {
		return false, ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if pw.Kind == k8sconsts.WorkloadKindStaticPod {
		if ic == nil {
			return false, ctrl.Result{}, nil
		}
		statusChanged := meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  string(odigosv1alpha1.WorkloadRolloutReasonWorkloadNotSupporting),
			Message: "static pods don't support restart",
		})
		return statusChanged, ctrl.Result{}, nil
	}

	if pw.Kind == k8sconsts.WorkloadKindCronJob || pw.Kind == k8sconsts.WorkloadKindJob {
		if ic == nil {
			return false, ctrl.Result{}, nil
		}
		statusChanged := meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  string(odigosv1alpha1.WorkloadRolloutReasonWaitingForRestart),
			Message: "Waiting for job to trigger by itself",
		})
		return statusChanged, ctrl.Result{}, nil
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

		if automaticRolloutDisabled {
			logger.Info("skipping rollout to uninstrument workload source - automatic rollout is disabled",
				"workload", pw.Name, "namespace", pw.Namespace)
			return false, ctrl.Result{}, nil
		}

		// instrumentation config is deleted, trigger a rollout for the associated workload
		// this should happen once per workload, as the instrumentation config is deleted
		// and we want to rollout the workload to remove the instrumentation
		rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
		return false, ctrl.Result{}, client.IgnoreNotFound(rolloutErr)
	}

	rollbackDisabled := false
	if conf.RollbackDisabled != nil {
		rollbackDisabled = *conf.RollbackDisabled
	}

	// check if at least one of the distributions used by this workload requires a rollout
	hasDistributionThatRequiresRollout := false
	for _, containerConfig := range ic.Spec.Containers {
		d := distroProvider.GetDistroByName(containerConfig.OtelDistroName)
		if d == nil {
			continue
		}
		if distro.IsRestartRequired(d, conf) {
			hasDistributionThatRequiresRollout = true
		}
	}

	if !hasDistributionThatRequiresRollout {
		// all distributions used by this workload do not require a restart
		// thus, no rollout is needed
		statusChanged := meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type: odigosv1alpha1.WorkloadRolloutStatusConditionType,
			// currently we interpret False as an error state, so we use True here to indicate a healthy state
			// it might be confusing since rollout is not actually done, but this is the closest match given the k8s condition semantics.
			Status:  metav1.ConditionTrue,
			Reason:  string(odigosv1alpha1.WorkloadRolloutReasonNotRequired),
			Message: "The selected instrumentation distributions do not require application restart",
		})
		return statusChanged, ctrl.Result{}, nil
	}

	rollbackGraceTime, _ := time.ParseDuration(consts.DefaultAutoRollbackGraceTime)
	if conf.RollbackGraceTime != "" {
		rollbackGraceTime, err = time.ParseDuration(conf.RollbackGraceTime)
		if err != nil {
			return false, ctrl.Result{}, fmt.Errorf("invalid duration %q: %w", rollbackGraceTime, err)
		}
	}

	rollbackStabilityWindow, _ := time.ParseDuration(consts.DefaultAutoRollbackStabilityWindow)
	if conf.RollbackStabilityWindow != "" {
		rollbackStabilityWindow, err = time.ParseDuration(conf.RollbackStabilityWindow)
		if err != nil {
			return false, ctrl.Result{}, fmt.Errorf("invalid duration %q: %w", rollbackGraceTime, err)
		}
	}

	if automaticRolloutDisabled {
		// TODO: add more fine grained status conditions for this case.
		// For example: if the workload has already been rolled out, we can set the status to true
		// and signal that the process is considered completed.
		// If manual rollout is required, we can mention this for better UX.
		statusChanged := meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
			Status:  metav1.ConditionTrue, // this might not be a success, need to refine into multiple discrete states
			Reason:  string(odigosv1alpha1.WorkloadRolloutReasonDisabled),
			Message: "odigos automatic rollout is disabled",
		})
		return statusChanged, ctrl.Result{}, nil
	}

	savedRolloutHash := ic.Status.WorkloadRolloutHash
	newRolloutHash := ic.Spec.AgentsMetaHash
	if savedRolloutHash == newRolloutHash {
		rolloutDone := utils.IsWorkloadRolloutDone(workloadObj)
		if !rolloutDone && !rollbackDisabled {
			backOffInfo, err := podBackOffDuration(ctx, c, workloadObj)
			if err != nil {
				logger.Error(err, "Failed to check pod backoff status")
				return false, ctrl.Result{}, err
			}

			if ic.Status.InstrumentationTime == nil {
				return false, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
			}
			now := time.Now()
			timeSinceInstrumentation := now.Sub(ic.Status.InstrumentationTime.Time)

			if backOffInfo.duration > 0 && timeSinceInstrumentation < rollbackStabilityWindow && ic.Spec.AgentInjectionEnabled {
				// Allow grace time for worklaod to stabelize before uninstrumenting it
				if backOffInfo.duration < rollbackGraceTime {
					return false, ctrl.Result{RequeueAfter: rollbackGraceTime - backOffInfo.duration}, nil
				}

				logger.Info("Triggering rollback due to backoff", "reason", backOffInfo.reason, "workload", pw.Name, "namespace", pw.Namespace)

				// Determine the reason based on which backoff was detected
				reason := backOffInfo.reason
				message := backOffInfo.message

				for i := range ic.Spec.Containers {
					ic.Spec.Containers[i].AgentEnabled = false
					ic.Spec.Containers[i].AgentEnabledReason = reason
				}
				ic.Spec.AgentInjectionEnabled = false

				if err := c.Update(ctx, ic); err != nil {
					res, err := utils.K8SUpdateErrorHandler(err)
					return false, res, err
				}

				ic.Status.RollbackOccurred = true

				rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
				if rolloutErr != nil {
					logger.Error(rolloutErr, "error rolling out workload", "name", pw.Name, "namespace", pw.Namespace)
					return false, ctrl.Result{}, rolloutErr
				}

				meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
					Type:    odigosv1alpha1.WorkloadRolloutStatusConditionType,
					Status:  metav1.ConditionTrue,
					Reason:  string(odigosv1alpha1.WorkloadRolloutReasonTriggeredSuccessfully),
					Message: message,
				})

				meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
					Type:    odigosv1alpha1.AgentEnabledStatusConditionType,
					Status:  metav1.ConditionFalse,
					Reason:  string(reason),
					Message: message,
				})
				// Status always changes, requeue to test wait for status change with workload
				return true, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
			}
			// Requeue to wait for workload to finish or enter backoff state
			return false, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
		}
		return false, ctrl.Result{}, nil
	}
	// if a rollout is ongoing, wait for it to finish, requeue
	statusChanged := false
	if !utils.IsWorkloadRolloutDone(workloadObj) {
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

	// If we have new rollout hash and also, AgentInjectionEnabled is enabled, that means we're instrumenting a new app
	if ic.Spec.AgentInjectionEnabled {
		now := metav1.NewTime(time.Now())
		ic.Status.InstrumentationTime = &now
	}
	meta.SetStatusCondition(&ic.Status.Conditions, rolloutCondition(rolloutErr))

	// at this point, the hashes are different, notify the caller the status has changed
	// Requeue to try and catch a crashing app
	return true, ctrl.Result{RequeueAfter: requeueWaitingForWorkloadRollout}, nil
}

// RolloutRestartWorkload restarts the given workload by patching its template annotations.
// this is bases on the kubectl implementation of restarting a workload
// https://github.com/kubernetes/kubectl/blob/master/pkg/polymorphichelpers/objectrestarter.go#L32
func rolloutRestartWorkload(ctx context.Context, workloadObj client.Object, c client.Client, ts time.Time) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, ts.Format(time.RFC3339)))
	switch obj := workloadObj.(type) {
	case *appsv1.Deployment:
		if obj.Spec.Paused {
			return errors.New("can't restart paused deployment")
		}
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
	case *appsv1.StatefulSet:
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
	case *appsv1.DaemonSet:
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
	case *openshiftappsv1.DeploymentConfig:
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
	case *argorolloutsv1alpha1.Rollout:
		// Rollouts use a different field (spec.restartAt) for restarting, so we need to patch it differently
		// https://github.com/argoproj/argo-rollouts/blob/cb1c33df7a2c2b1c2ed31b1ee0aa22621ef5577c/utils/replicaset/replicaset.go#L223-L232
		rolloutPatch := []byte(fmt.Sprintf(`{"spec":{"restartAt":"%s"}}`, ts.Format(time.RFC3339)))
		return c.Patch(ctx, obj, client.RawPatch(types.MergePatchType, rolloutPatch))
	case *corev1.Pod:
		if workload.IsStaticPod(obj) {
			return errors.New("can't restart static pods")
		}
		return errors.New("currently not supporting standalone pods as workloads for rollout")
	default:
		return errors.New("unknown kind")
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

// podBackOffInfo contains information about pod backoff state
type podBackOffInfo struct {
	duration time.Duration
	reason   odigosv1alpha1.AgentEnabledReason
	message  string
}

// podHasBackOff returns true if any (init)-container in the pod is in CrashLoopBackOff or ImagePullBackOff.
func podHasBackOff(p *corev1.Pod) bool {
	allStatuses := append(p.Status.InitContainerStatuses, p.Status.ContainerStatuses...)
	for _, cs := range allStatuses {
		if containerutils.IsContainerInBackOff(&cs) {
			return true
		}
	}
	return false
}

// getPodBackOffReason returns the backoff reason for a pod, prioritizing CrashLoopBackOff over ImagePullBackOff
func getPodBackOffReason(p *corev1.Pod) (odigosv1alpha1.AgentEnabledReason, string) {
	for _, cs := range append(p.Status.InitContainerStatuses, p.Status.ContainerStatuses...) {
		if containerutils.IsContainerInCrashLoopBackOff(&cs) {
			return odigosv1alpha1.AgentEnabledReasonCrashLoopBackOff, "pods entered CrashLoopBackOff; instrumentation disabled"
		}
		if containerutils.IsContainerInImagePullBackOff(&cs) {
			return odigosv1alpha1.AgentEnabledReasonImagePullBackOff, "pods entered ImagePullBackOff; instrumentation disabled"
		}
	}
	return "", ""
}

// instrumentedPodsSelector returns a selector for all the instrumented pods that are associated with the workload object
func instrumentedPodsSelector(obj client.Object) (labels.Selector, error) {
	var selector *metav1.LabelSelector

	switch o := obj.(type) {
	case *appsv1.Deployment:
		selector = o.Spec.Selector
	case *appsv1.StatefulSet:
		selector = o.Spec.Selector
	case *appsv1.DaemonSet:
		selector = o.Spec.Selector
	case *openshiftappsv1.DeploymentConfig:
		// DeploymentConfig selector is map[string]string, convert to *metav1.LabelSelector
		selector = &metav1.LabelSelector{
			MatchLabels: o.Spec.Selector,
		}
	case *argorolloutsv1alpha1.Rollout:
		selector = &metav1.LabelSelector{
			MatchLabels: o.Spec.Selector.MatchLabels,
		}
	default:
		return nil, fmt.Errorf("crashLoopBackOffDuration: unsupported workload kind %T", obj)
	}

	if selector == nil {
		return nil, fmt.Errorf("crashLoopBackOffDuration: workload has nil selector")
	}

	// Create a deep copy of the selector to avoid mutating the original
	selectorCopy := selector.DeepCopy()
	selectorCopy.MatchExpressions = append(selectorCopy.MatchExpressions, metav1.LabelSelectorRequirement{
		Key:      k8sconsts.OdigosAgentsMetaHashLabel,
		Operator: metav1.LabelSelectorOpExists,
	})
	sel, err := metav1.LabelSelectorAsSelector(selectorCopy)
	if err != nil {
		return nil, fmt.Errorf("crashLoopBackOffDuration: invalid selector: %w", err)
	}

	return sel, nil
}

// podBackOffDuration returns how long the supplied workload
// (Deployment, StatefulSet, or DaemonSet) has been in CrashLoopBackOff or ImagePullBackOff.
//
// It inspects all Pods selected by the workload's label selector:
//
//   - If at least one Pod is currently in CrashLoopBackOff or ImagePullBackOff, it finds the
//     earliest Pod.StartTime among those Pods and returns the elapsed time
//     since that moment, along with the reason.
//
//   - If **no** Pod is in backoff state, it simply returns 0 duration and no error.
//
// A non-nil error is returned only for unexpected situations (e.g. unsupported
// workload kind, invalid selector, or failed Pod list call).
func podBackOffDuration(ctx context.Context, c client.Client, obj client.Object) (podBackOffInfo, error) {
	sel, err := instrumentedPodsSelector(obj)
	if err != nil {
		return podBackOffInfo{}, fmt.Errorf("podBackOffDuration: invalid selector: %w", err)
	}

	// 2. List matching Pods once (single API call for both checks).
	var podList corev1.PodList
	if err := c.List(ctx, &podList,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		return podBackOffInfo{}, fmt.Errorf("podBackOffDuration: failed listing pods: %w", err)
	}

	// 3. Find the earliest-started Pod that is in backoff state.
	var earliest *time.Time
	var earliestReason odigosv1alpha1.AgentEnabledReason
	var earliestMessage string
	for i := range podList.Items {
		p := &podList.Items[i]

		if !podHasBackOff(p) {
			continue
		}
		if p.Status.StartTime == nil { // extremely rare
			continue
		}

		start := p.Status.StartTime.Time
		if earliest == nil || start.Before(*earliest) {
			earliest = &start
			earliestReason, earliestMessage = getPodBackOffReason(p)
		}
	}

	// 4. Return zero duration if nothing is in backoff state.
	if earliest == nil {
		return podBackOffInfo{}, nil
	}

	// 5. Otherwise, duration since the workload entered backoff state.
	return podBackOffInfo{
		duration: time.Since(*earliest),
		reason:   earliestReason,
		message:  earliestMessage,
	}, nil
}

// workloadHasOdigosAgents returns true if the workload still has *any* pod present in the instrumented-pod.
func workloadHasOdigosAgents(ctx context.Context, c client.Client, obj client.Object) (bool, error) {
	sel, err := instrumentedPodsSelector(obj)
	if err != nil {
		return false, fmt.Errorf("workloadHasOdigosAgents: invalid selector: %w", err)
	}

	var pods corev1.PodList
	if err := c.List(
		ctx, &pods,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		return false, fmt.Errorf("workloadHasOdigosAgents: listing pods failed: %w", err)
	}

	// any non-empty list means the workload still runs instrumented pods.
	return len(pods.Items) > 0, nil
}
