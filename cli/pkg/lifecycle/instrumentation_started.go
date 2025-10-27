package lifecycle

import (
	"context"

	"github.com/odigos-io/odigos/cli/pkg/remote"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstrumentationStarted struct {
	BaseTransition
}

func (i *InstrumentationStarted) From() State {
	return StateLangDetected
}

func (i *InstrumentationStarted) To() State {
	return StateInstrumentationInProgress
}

func (i *InstrumentationStarted) Execute(ctx context.Context, obj metav1.Object) error {
	// Instrumentation is started already, nothing to execute here. This transition just verifies that the pods are running.
	return nil
}

func (i *InstrumentationStarted) GetTransitionState(ctx context.Context, obj metav1.Object) (State, error) {
	kind := WorkloadKindFrombject(obj)
	if !i.remote {
		icName := workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), kind)
		ic, err := i.client.OdigosClient.InstrumentationConfigs(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				i.log("Error while fetching InstrumentationConfig: " + err.Error())
				return StateUnknown, err
			}
			return i.From(), nil
		}

		if len(ic.Spec.Containers) == 0 {
			return i.From(), nil
		}
	} else {
		describe, err := remote.DescribeSource(ctx, i.client, obj.GetNamespace(), string(kind), obj.GetNamespace(), obj.GetName())
		if err != nil {
			return StateUnknown, err
		}
		if describe.OtelAgents.Containers == nil || len(describe.OtelAgents.Containers) == 0 {
			return i.From(), nil
		}
	}

	return i.To(), nil
}

var _ Transition = &InstrumentationStarted{}
