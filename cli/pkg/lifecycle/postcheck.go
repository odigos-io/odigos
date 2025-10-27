package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type PostCheck struct {
	BaseTransition
}

func (i *PostCheck) From() State {
	return StateInstrumented
}

func (i *PostCheck) To() State {
	return StatePostCheckPassed
}

func (i *PostCheck) Execute(ctx context.Context, obj metav1.Object) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Minute, true, func(ctx context.Context) (bool, error) {
		i.log("Waiting for all pods to be healthy ...")
		rolloutCompleted, err := utils.VerifyAllPodsAreRunning(ctx, i.client, obj)
		if err != nil {
			i.log("Error verifying all pods are healthy")
			return false, err
		}

		if rolloutCompleted {
			i.log("Rollout completed, all running pods are healthy")
			coolOff := GetCoolOff(ctx)
			if coolOff > 0 {
				i.log("Cool off flag is set, waiting for pods to be Running for " + coolOff.String() + " before marking the workload as instrumented")
				time.Sleep(coolOff)
				afterCoolOff, err := utils.VerifyAllPodsAreRunning(ctx, i.client, obj)
				if err != nil {
					i.log("Error verifying all pods are healthy")
					return false, err
				}

				if afterCoolOff {
					i.log("Cool off check completed, all running pods are healthy")
					return true, nil
				} else {
					i.log("Cool off check not completed, all running pods are not healthy")
					return false, nil
				}
			} else {
				return true, nil
			}
		} else {
			podsInPhase := make(map[string]int)
			labelSelector := &metav1.LabelSelector{MatchLabels: utils.GetMatchLabels(obj)}
			podList, err := i.client.CoreV1().Pods(obj.GetNamespace()).List(ctx, metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(labelSelector),
			})
			if err != nil {
				return false, err
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

func (i *PostCheck) GetTransitionState(ctx context.Context, obj metav1.Object) (State, error) {
	rolloutCompleted, err := utils.VerifyAllPodsAreRunning(ctx, i.client, obj)
	if err != nil {
		i.log("Error verifying all pods are instrumented")
		return StateUnknown, err
	}
	if !rolloutCompleted {
		return i.From(), nil
	}
	return i.To(), nil
}

var _ Transition = &PostCheck{}
