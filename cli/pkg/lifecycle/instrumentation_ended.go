package lifecycle

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/pkg/remote"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type InstrumentationEnded struct {
	BaseTransition
}

func (i *InstrumentationEnded) From() State {
	return StateInstrumentationInProgress
}

func (i *InstrumentationEnded) To() State {
	return StateInstrumented
}

func (i *InstrumentationEnded) Execute(ctx context.Context, obj metav1.Object) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Minute, true, func(ctx context.Context) (bool, error) {
		return i.allInstrumentedPodsAreRunning(ctx, obj)
	})
}

func (i *InstrumentationEnded) GetTransitionState(ctx context.Context, obj metav1.Object) (State, error) {
	rolloutCompleted, err := i.allInstrumentedPodsAreRunning(ctx, obj)
	if err != nil {
		return StateUnknown, err
	}
	if !rolloutCompleted {
		return i.From(), nil
	}
	return i.To(), nil
}

var _ Transition = &InstrumentationEnded{}

func (i *InstrumentationEnded) allInstrumentedPodsAreRunning(ctx context.Context, obj metav1.Object) (bool, error) {
	if !i.remote {
		// Print number of pods per phase
		podsInPhase := make(map[string]int)
		labelSelector := &metav1.LabelSelector{MatchLabels: utils.GetMatchLabels(obj)}
		podList, err := i.client.CoreV1().Pods(obj.GetNamespace()).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(labelSelector),
		})
		if err != nil {
			return false, err
		}

		instrumentedLabelSelector := &metav1.LabelSelector{
			MatchLabels: utils.GetMatchLabels(obj),
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      k8sconsts.OdigosAgentsMetaHashLabel,
					Operator: metav1.LabelSelectorOpExists,
				},
			},
		}

		instrumentedPodList, err := i.client.CoreV1().Pods(obj.GetNamespace()).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(instrumentedLabelSelector),
		})
		if err != nil {
			return false, err
		}

		for _, pod := range instrumentedPodList.Items {
			podsInPhase[string(pod.Status.Phase)]++
		}

		// Print how many pods in every phase in one line
		i.log(fmt.Sprintf("Total pods: %d, Instrumented: %d, Instrumented and Running: %d", len(podList.Items), len(instrumentedPodList.Items), podsInPhase[string(v1.PodRunning)]))

		// If all instrumented pods are running, return true
		if podsInPhase[string(v1.PodRunning)] == len(instrumentedPodList.Items) {
			return true, nil
		}
		return false, nil
	} else {
		des, err := remote.DescribeSource(ctx, i.client, i.odigosNamespace, string(WorkloadKindFrombject(obj)), obj.GetNamespace(), obj.GetName())
		if err != nil {
			i.log("Error describing pod")
			return false, err
		}

		instrumentedPods := 0
		instrumentedAndRunningPods := 0
		for _, pod := range des.Pods {
			if pod.AgentInjected.Value == true {
				instrumentedPods++
				if pod.Phase.Value == string(v1.PodRunning) {
					instrumentedAndRunningPods++
				}
			}
		}
		i.log(fmt.Sprintf("Total Pods: %d, Instrumented: %d, Instrumented and Running %d", des.TotalPods, instrumentedPods, instrumentedAndRunningPods))
		if instrumentedAndRunningPods == instrumentedPods {
			return true, nil
		}
		return false, nil
	}
}
