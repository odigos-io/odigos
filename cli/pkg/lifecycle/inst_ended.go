package lifecycle

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationEnded struct {
	BaseTransition
}

func (i *InstrumentationEnded) From() State {
	return InstrumentationInProgress
}

func (i *InstrumentationEnded) To() State {
	return InstrumentedState
}

func (i *InstrumentationEnded) Execute(ctx context.Context, obj client.Object, templateSpec *corev1.PodTemplateSpec, isRemote bool) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Minute, true, func(ctx context.Context) (bool, error) {
		i.log("Waiting for all pods to be instrumented ...")
		rolloutCompleted, err := utils.VerifyAllPodsAreInstrumented(ctx, i.client, obj)
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
				afterCoolOff, err := utils.VerifyAllPodsAreInstrumented(ctx, i.client, obj)
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
			podList, err := i.client.CoreV1().Pods(obj.GetNamespace()).List(ctx, metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(
					&metav1.LabelSelector{MatchLabels: utils.GetMatchLabels(obj)}),
			})
			if err != nil {
				return false, nil
			}

			for _, pod := range podList.Items {
				podsInPhase[string(pod.Status.Phase)]++
			}

			// Print how many pods in every phase in one line
			i.log(fmt.Sprintf("Pods status: %v", podsInPhase))
		}

		return false, nil
	})
}
