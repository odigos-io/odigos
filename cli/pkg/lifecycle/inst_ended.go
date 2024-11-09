package lifecycle

import (
	"context"
	"time"

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

func (i *InstrumentationEnded) Execute(ctx context.Context, obj client.Object, templateSpec *corev1.PodTemplateSpec) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Minute, true, func(ctx context.Context) (bool, error) {
		rolloutCompleted, err := utils.VerifyAllPodsAreInstrumented(ctx, i.client, obj)
		if err != nil {
			i.log("Error verifying all pods are instrumented")
			return false, err
		}

		if rolloutCompleted {
			i.log("Rollout completed, all running pods contains the new instrumentation")
		}

		return rolloutCompleted, nil
	})
}
