package rollout

import (
	"context"
	"errors"
	"fmt"
	"time"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
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

const RequeueWaitingForWorkloadRollout = 10 * time.Second

type RolloutResult struct {
	StatusChanged bool
	// Result contains the controller result for requeue behavior.
	Result ctrl.Result
}

// WorkloadKey generates a unique key for rate limiting purposes
func WorkloadKey(pw k8sconsts.PodWorkload) string {
	return fmt.Sprintf("%s/%s/%s", pw.Namespace, pw.Kind, pw.Name)
}

// Do potentially triggers a rollout for the given workload based on the given instrumentation config.
// If the instrumentation config is nil, the workload is rolled out - this is used for un-instrumenting workloads.
// Otherwise, the rollout config hash is calculated and compared to the saved hash in the instrumentation config.
// If the hashes are different, the workload is rolled out.
// If the hashes are the same, this is a no-op.
//
// If a rollout is triggered the status of the instrumentation config is updated with the new rollout hash
// and a corresponding condition is set.
//
// Returns a RolloutResult and an error.
func Do(ctx context.Context, c client.Client, ic *odigosv1alpha1.InstrumentationConfig, pw k8sconsts.PodWorkload, conf *common.OdigosConfiguration, distroProvider *distros.Provider, rolloutConcurrencyLimiter *RolloutConcurrencyLimiter) (RolloutResult, error) {
	isAutomaticRolloutDisabled, rollBackOptions, configErr := getRolloutAndRollbackOptions(conf)
	if configErr != nil {
		return RolloutResult{}, configErr
	}
	logger := log.FromContext(ctx)
	workloadObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	getErr := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, workloadObj)
	if getErr != nil {
		return RolloutResult{}, client.IgnoreNotFound(getErr)
	}

	// Don't allow rollout of static pods, cronjobs or jobs
	if pw.Kind == k8sconsts.WorkloadKindStaticPod {
		if ic == nil {
			return RolloutResult{}, nil
		}
		changed := meta.SetStatusCondition(&ic.Status.Conditions, conditionStaticPodsNotSupported)
		return RolloutResult{StatusChanged: changed}, nil
	}

	if pw.Kind == k8sconsts.WorkloadKindCronJob || pw.Kind == k8sconsts.WorkloadKindJob {
		if ic == nil {
			return RolloutResult{}, nil
		}
		changed := meta.SetStatusCondition(&ic.Status.Conditions, conditionWaitingForJobTrigger)
		return RolloutResult{StatusChanged: changed}, nil
	}

	if ic == nil {
		// If ic is nil and the PodWorkload is missing the odigos.io/agents-meta-hash label,
		// it means it is a rolled back application that shouldn't be rolled out again.
		hasAgents, agentsErr := workloadHasOdigosAgents(ctx, c, workloadObj)
		if agentsErr != nil {
			logger.Error(agentsErr, "failed to check for odigos agent labels")
			return RolloutResult{}, agentsErr
		}
		if !hasAgents {
			logger.Info("skipping rollout - workload already runs without odigos agents",
				"workload", pw.Name, "namespace", pw.Namespace)
			return RolloutResult{}, nil
		}

		if isAutomaticRolloutDisabled {
			logger.Info("skipping rollout to uninstrument workload source - automatic rollout is disabled",
				"workload", pw.Name, "namespace", pw.Namespace)
			return RolloutResult{}, nil
		}

		// instrumentation config is deleted, trigger a rollout for the associated workload
		// this should happen once per workload, as the instrumentation config is deleted
		// and we want to rollout the workload to remove the instrumentation
		// Note: uninstrumentation rollouts are not rate limited since we can't track completion
		// (the IC is deleted so we won't get subsequent reconciles)
		logger.V(2).Info("proceeding with uninstrumentation rollout",
			"workload", pw.Name,
			"namespace", pw.Namespace)
		rolloutConcurrencyLimiter.ReleaseWorkloadRolloutSlot(WorkloadKey(pw))
		rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
		return RolloutResult{}, client.IgnoreNotFound(rolloutErr)
	}

	// Check if recovery from rollback is needed before proceeding with rollout logic.
	// If recovery is needed, persist the changes and requeue so the next reconcile
	// recomputes the spec with RollbackOccurred cleared.
	if recoverFromRollback(ic) {
		if err := c.Update(ctx, ic); err != nil {
			result, handledErr := utils.K8SUpdateErrorHandler(err)
			return RolloutResult{Result: result}, handledErr
		}
		// c.Update refreshes the in-memory object, overwriting status changes. Re-apply and persist.
		ic.Status.RollbackOccurred = false
		if err := c.Status().Update(ctx, ic); err != nil {
			result, handledErr := utils.K8SUpdateErrorHandler(err)
			return RolloutResult{Result: result}, handledErr
		}
		return RolloutResult{Result: ctrl.Result{Requeue: true}}, nil
	}

	if ic.Spec.PodManifestInjectionOptional {
		// all distributions used by this workload do not require a restart
		// thus, no rollout is needed
		rolloutConcurrencyLimiter.ReleaseWorkloadRolloutSlot(WorkloadKey(pw))
		changed := meta.SetStatusCondition(&ic.Status.Conditions, conditionRestartNotRequiredForDistro)
		return RolloutResult{StatusChanged: changed}, nil
	}

	if isAutomaticRolloutDisabled {
		// TODO: add more fine grained status conditions for this case.
		// For example: if the workload has already been rolled out, we can set the status to true
		// and signal that the process is considered completed.
		// If manual rollout is required, we can mention this for better UX.
		rolloutConcurrencyLimiter.ReleaseWorkloadRolloutSlot(WorkloadKey(pw))
		changed := meta.SetStatusCondition(&ic.Status.Conditions, conditionRolloutDisabled)
		return RolloutResult{StatusChanged: changed}, nil
	}

	workloadKey := WorkloadKey(pw)
	savedRolloutHash := ic.Status.WorkloadRolloutHash
	newRolloutHash := ic.Spec.AgentsMetaHash
	// Scenario: successful instrumentation ("X"="X") or successful uninstrumentation (""="")
	if savedRolloutHash == newRolloutHash {
		rolloutDone := utils.IsWorkloadRolloutDone(workloadObj)

		if !rolloutDone && !rollBackOptions.IsRollbackDisabled {
			// Rollback scenario: already instrumented workload is in backoff state, trigger rollback.
			shouldRollback, waitDuration, backOffInfo, err := shouldTriggerRollback(
				ctx, c, ic, workloadObj, rollBackOptions,
			)
			if err != nil {
				logger.Error(err, "Failed to check pod backoff status")
				return RolloutResult{}, err
			}
			if waitDuration > 0 && shouldRollback {
				return RolloutResult{Result: ctrl.Result{RequeueAfter: waitDuration}}, nil
			}
			if shouldRollback {
				return triggerRollback(ctx, c, logger, ic, workloadObj, workloadKey,
					rolloutConcurrencyLimiter, &backOffInfo, pw)
			}

			// Requeue to wait for workload to finish or enter backoff state
			return RolloutResult{Result: ctrl.Result{RequeueAfter: RequeueWaitingForWorkloadRollout}}, nil
		}

		// at this point, we know that rollout happened.
		// make sure to cleanup any stale condition from previous rollout attempts.
		// example:
		// - workload is instrumented successfully.
		// - rollout regardless of odigos
		// - agent disabled -> status is changed to "waiting for previous rollout to finish"
		// - agent enabled -> no need for a rollout, but the status needs to be updated.
		statusChanged := false
		// This is the happy flow - the workload is rolled out successfully
		if rolloutDone {
			statusChanged = meta.SetStatusCondition(&ic.Status.Conditions, conditionRolloutFinished)
			// Rollout is complete - release the slot if we had one
			rolloutConcurrencyLimiter.ReleaseWorkloadRolloutSlot(workloadKey)
		}
		return RolloutResult{StatusChanged: statusChanged, Result: ctrl.Result{}}, nil
	}

	// Rollback scenario: Webhook-instrumented pod crashlooping before rollout: AgentsMetaHash is populated but WorkloadRolloutHash is empty.
	if savedRolloutHash == "" && newRolloutHash != "" &&
		!rollBackOptions.IsRollbackDisabled && ic.Spec.AgentInjectionEnabled {
		// TODO: this must be changed - if rate limiting is enabled, then this time will prevent webhook crashlooping pods from rollbacking.
		// This is becasue the AgentsMetaHashChangedTime is set when the pod is created by the agentsenabled, but rate limiting may cause the actual rollout
		// to be much further on, preventing rollback when neccessary.
		ic.Status.InstrumentationTime = ic.Spec.AgentsMetaHashChangedTime

		shouldRollback, waitDuration, backOffInfo, err := shouldTriggerRollback(
			ctx, c, ic, workloadObj, rollBackOptions,
		)
		if err != nil {
			logger.Error(err, "Failed to check pod backoff status for pre-rollout pods")
			return RolloutResult{}, err
		}
		if waitDuration > 0 && shouldRollback {
			return RolloutResult{Result: ctrl.Result{RequeueAfter: waitDuration}}, nil
		}
		if shouldRollback {
			return triggerRollback(ctx, c, logger, ic, workloadObj, workloadKey,
				rolloutConcurrencyLimiter, &backOffInfo, pw)
		}
	}

	// Rollback scenario: new instrumented pod was inited due to new runtime, but it is potentially crashlooping, check for rollback
	if savedRolloutHash != "" && newRolloutHash != "" && savedRolloutHash != newRolloutHash &&
		!rollBackOptions.IsRollbackDisabled && ic.Spec.AgentInjectionEnabled {
		shouldRollback, waitDuration, backOffInfo, err := shouldTriggerRollback(
			ctx, c, ic, workloadObj, rollBackOptions,
		)
		if err != nil {
			logger.Error(err, "Failed to check pod backoff status pods with new runtime")
			return RolloutResult{}, err
		}
		if waitDuration > 0 && shouldRollback {
			return RolloutResult{Result: ctrl.Result{RequeueAfter: waitDuration}}, nil
		}
		if shouldRollback {
			return triggerRollback(ctx, c, logger, ic, workloadObj, workloadKey,
				rolloutConcurrencyLimiter, &backOffInfo, pw)
		}
	}

	// if a rollout is ongoing, wait for it to finish, requeue
	statusChanged := false
	if !utils.IsWorkloadRolloutDone(workloadObj) {
		statusChanged = meta.SetStatusCondition(&ic.Status.Conditions, conditionPreviousRolloutOngoing)
		return RolloutResult{StatusChanged: statusChanged, Result: ctrl.Result{RequeueAfter: RequeueWaitingForWorkloadRollout}}, nil
	}

	if !rolloutConcurrencyLimiter.TryAcquire(workloadKey, rollBackOptions.MaxConcurrentRollouts) {
		logger.V(2).Info("rate limited instrumentation rollout, requeuing",
			"workload", pw.Name,
			"namespace", pw.Namespace,
			"requeueAfter", RequeueWaitingForWorkloadRollout)
		statusChanged = meta.SetStatusCondition(&ic.Status.Conditions, conditionWaitingInQueue)
		return RolloutResult{StatusChanged: statusChanged, Result: ctrl.Result{RequeueAfter: RequeueWaitingForWorkloadRollout}}, nil
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
	// Setting the condition for successful triggering the rollout, or a failed to patch condition with a specific error message.
	meta.SetStatusCondition(&ic.Status.Conditions, rolloutCondition(rolloutErr))

	// at this point, the hashes are different, notify the caller the status has changed
	// Requeue to try and catch a crashing app
	return RolloutResult{StatusChanged: true, Result: ctrl.Result{RequeueAfter: RequeueWaitingForWorkloadRollout}}, nil
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
	if rolloutErr == nil {
		return conditionTriggeredSuccessfully
	}
	return newConditionFailedToPatch(rolloutErr)
}

// WorkloadHasNonInstrumentedPodInBackoff checks if any NON-INSTRUMENTED pod belonging to the workload is in a backoff state
// (CrashLoopBackOff or ImagePullBackOff). This specifically checks pods that DON'T have the odigos label,
// to detect pre-existing crashloops and prevent attempting to instrument already-crashlooping workloads.
// Pods that are already instrumented (have the odigos label) are handled by the rollback logic instead.
func WorkloadHasNonInstrumentedPodInBackoff(ctx context.Context, c client.Client, workloadObj client.Object) (bool, error) {
	if pod, ok := workloadObj.(*corev1.Pod); ok {
		return podHasBackOff(pod), nil
	}

	selector, err := notInstrumentedWorkloadPodsSelector(workloadObj)
	if err != nil {
		return false, fmt.Errorf("WorkloadHasNonInstrumentedPodInBackoff: %w", err)
	}

	var podList corev1.PodList
	if err := c.List(ctx, &podList,
		client.InNamespace(workloadObj.GetNamespace()),
		client.MatchingLabelsSelector{Selector: selector},
	); err != nil {
		return false, fmt.Errorf("WorkloadHasNonInstrumentedPodInBackoff: failed listing pods: %w", err)
	}

	// If a pod has the odigos label and is in backoff, it's handled by the rollback logic.
	for i := range podList.Items {
		pod := &podList.Items[i]
		if podHasBackOff(pod) {
			return true, nil
		}
	}

	return false, nil
}

// workloadLabelSelector returns the label selector for a workload object
func workloadLabelSelector(obj client.Object) (*metav1.LabelSelector, error) {
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
		return nil, fmt.Errorf("workloadLabelSelector: unsupported workload kind %T", obj)
	}

	if selector == nil {
		return nil, fmt.Errorf("workloadLabelSelector: workload has nil selector")
	}

	return selector, nil
}

// workloadPodsSelector returns a selector for all pods that are associated with the workload object
// who do not have the odigos instrumentation label
func notInstrumentedWorkloadPodsSelector(obj client.Object) (labels.Selector, error) {
	labelSelector, err := workloadLabelSelector(obj)
	if err != nil {
		return nil, err
	}

	selectorCopy := labelSelector.DeepCopy()
	selectorCopy.MatchExpressions = append(selectorCopy.MatchExpressions, metav1.LabelSelectorRequirement{
		Key:      k8sconsts.OdigosAgentsMetaHashLabel,
		Operator: metav1.LabelSelectorOpDoesNotExist,
	})

	selector, err := metav1.LabelSelectorAsSelector(selectorCopy)
	if err != nil {
		return nil, fmt.Errorf("notInstrumentedWorkloadPodsSelector: invalid selector: %w", err)
	}

	return selector, nil
}

// instrumentedPodsSelector returns a selector for all the instrumented pods that are associated with the workload object
func instrumentedPodsSelector(obj client.Object) (labels.Selector, error) {
	labelSelector, err := workloadLabelSelector(obj)
	if err != nil {
		return nil, err
	}

	// Create a deep copy of the selector to avoid mutating the original
	selectorCopy := labelSelector.DeepCopy()
	selectorCopy.MatchExpressions = append(selectorCopy.MatchExpressions, metav1.LabelSelectorRequirement{
		Key:      k8sconsts.OdigosAgentsMetaHashLabel,
		Operator: metav1.LabelSelectorOpExists,
	})

	sel, err := metav1.LabelSelectorAsSelector(selectorCopy)
	if err != nil {
		return nil, fmt.Errorf("instrumentedPodsSelector: invalid selector: %w", err)
	}

	return sel, nil
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

// shouldTriggerRollback checks if rollback should be triggered based on backoff state and timing.
// Original condition: backOffInfo.duration > 0 && timeSinceInstrumentation < RollbackStabilityWindow && AgentInjectionEnabled
// Returns:
// - shouldRollback=true, waitDuration > 0: grace period not elapsed, requeue and wait
// - shouldRollback=true, waitDuration=0: execute rollback now
// - shouldRollback=false: no rollback needed
func shouldTriggerRollback(
	ctx context.Context,
	c client.Client,
	ic *odigosv1alpha1.InstrumentationConfig,
	workloadObj client.Object,
	rollBackOptions RollBackOptions,
) (shouldRollback bool, waitDuration time.Duration, backOffInfo podBackOffInfo, err error) {
	backOffInfo, err = podBackOffDuration(ctx, c, workloadObj)
	if err != nil {
		return false, 0, podBackOffInfo{}, err
	}

	if ic.Status.InstrumentationTime == nil {
		return false, RequeueWaitingForWorkloadRollout, podBackOffInfo{}, nil
	}

	timeSinceInstrumentation := time.Since(ic.Status.InstrumentationTime.Time)

	// Original condition: backOffInfo.duration > 0 && timeSinceInstrumentation < RollbackStabilityWindow && AgentInjectionEnabled
	if backOffInfo.duration <= 0 || timeSinceInstrumentation >= rollBackOptions.RollbackStabilityWindow || !ic.Spec.AgentInjectionEnabled {
		return false, 0, podBackOffInfo{}, nil
	}

	// At this point, rollback conditions are met (the outer condition from original code)
	// Allow grace time for workload to stabilize before uninstrumenting it
	if backOffInfo.duration < rollBackOptions.RollbackGraceTime {
		return true, rollBackOptions.RollbackGraceTime - backOffInfo.duration, backOffInfo, nil
	}

	return true, 0, backOffInfo, nil
}

// recoverFromRollback checks if a rollback recovery was requested by comparing the
// RollbackRecoveryAtAnnotation (desired, propagated from Source) with the
// RollbackRecoveryProcessedAtAnnotation (last processed). If they differ, it clears
// RollbackOccurred and updates the processed annotation.
// Returns true if recovery was applied and the IC was modified.
func recoverFromRollback(ic *odigosv1alpha1.InstrumentationConfig) bool {
	currentRecoveryAt := ic.Annotations[k8sconsts.RollbackRecoveryAtAnnotation]
	if !ic.Status.RollbackOccurred || currentRecoveryAt == "" {
		return false
	}

	processedRecoveryAt := ic.Annotations[k8sconsts.RollbackRecoveryProcessedAtAnnotation]
	if processedRecoveryAt == currentRecoveryAt {
		return false
	}

	if ic.Annotations == nil {
		ic.Annotations = make(map[string]string)
	}
	ic.Annotations[k8sconsts.RollbackRecoveryProcessedAtAnnotation] = currentRecoveryAt
	return true
}

// triggerRollback executes the rollback: disables agents, updates IC, and restarts the workload.
func triggerRollback(
	ctx context.Context,
	c client.Client,
	logger logr.Logger,
	ic *odigosv1alpha1.InstrumentationConfig,
	workloadObj client.Object,
	workloadKey string,
	rolloutConcurrencyLimiter *RolloutConcurrencyLimiter,
	backOffInfo *podBackOffInfo,
	pw k8sconsts.PodWorkload,
) (RolloutResult, error) {
	logger.Info("Triggering rollback due to backoff",
		"reason", backOffInfo.reason,
		"workload", pw.Name,
		"namespace", pw.Namespace)

	for i := range ic.Spec.Containers {
		ic.Spec.Containers[i].AgentEnabled = false
		ic.Spec.Containers[i].AgentEnabledReason = backOffInfo.reason
	}
	ic.Spec.AgentInjectionEnabled = false

	if err := c.Update(ctx, ic); err != nil {
		_, handledErr := utils.K8SUpdateErrorHandler(err)
		return RolloutResult{}, handledErr
	}

	ic.Status.RollbackOccurred = true

	// Release any rate limiter slot (rollback bypasses rate limiting)
	rolloutConcurrencyLimiter.ReleaseWorkloadRolloutSlot(workloadKey)

	// Restart the workload to remove instrumentation
	rolloutErr := rolloutRestartWorkload(ctx, workloadObj, c, time.Now())
	if rolloutErr != nil {
		logger.Error(rolloutErr, "error rolling out workload", "name", pw.Name, "namespace", pw.Namespace)
		return RolloutResult{}, rolloutErr
	}

	meta.SetStatusCondition(&ic.Status.Conditions, newConditionTriggeredWithMessage(backOffInfo.message))
	meta.SetStatusCondition(&ic.Status.Conditions, newConditionAgentDisabledDueToBackoff(backOffInfo.reason, backOffInfo.message))

	return RolloutResult{StatusChanged: true, Result: ctrl.Result{RequeueAfter: RequeueWaitingForWorkloadRollout}}, nil
}
