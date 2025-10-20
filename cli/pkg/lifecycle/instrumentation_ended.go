package lifecycle

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/odigos-io/odigos/api/k8sconsts"
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
		i.log("Waiting for all pods to be instrumented ...")
		rolloutCompleted, err := utils.VerifyAllPodsAreRunning(ctx, i.client, obj, true)
		if err != nil {
			i.log("Error verifying all pods are instrumented")
			return false, err
		}

		if rolloutCompleted {
			i.log("Rollout completed, all running pods contains the new instrumentation")
			coolOff := GetCoolOff(ctx)
			if coolOff > 0 {
				i.log("Cool off flag is set, waiting for pods to be Running for " + coolOff.String() + " before marking the workload as instrumented")
				time.Sleep(coolOff)
				afterCoolOff, err := utils.VerifyAllPodsAreRunning(ctx, i.client, obj, true)
				if err != nil {
					i.log("Error verifying all pods are instrumented")
					return false, err
				}

				if afterCoolOff {
					i.log("Cool off check completed, all running pods contains the new instrumentation")
					return true, nil
				} else {
					i.log("Cool off check not completed, all running pods does not contain the new instrumentation")
					return false, nil
				}
			} else {
				return true, nil
			}
		} else {
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
			i.log(fmt.Sprintf("Pods status: Total: %d, Instrumented: %v", len(podList.Items), podsInPhase))
		}

		return false, nil
	})
}

func (i *InstrumentationEnded) GetTransitionState(ctx context.Context, obj metav1.Object) (State, error) {
	rolloutCompleted, err := utils.VerifyAllPodsAreRunning(ctx, i.client, obj, true)
	if err != nil {
		i.log("Error verifying all pods are instrumented")
		return StateUnknown, err
	}
	if !rolloutCompleted {
		return i.From(), nil
	}
	return i.To(), nil
}

var _ Transition = &InstrumentationEnded{}
