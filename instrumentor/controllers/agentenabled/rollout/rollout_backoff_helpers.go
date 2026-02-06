package rollout

import (
	"context"
	"fmt"
	"time"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	containerutils "github.com/odigos-io/odigos/k8sutils/pkg/container"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
