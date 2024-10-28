package lifecycle

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/cli/cmd/resources"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	v1 "k8s.io/api/core/v1"
)

type State string

const (
	UnknownState              State = "Unknown"
	NotInstrumentedState      State = "NotInstrumented"
	PreflightChecksPassed     State = "PreflightChecksPassed"
	LangDetectionInProgress   State = "LangDetectionInProgress"
	LangDetectedState         State = "LangDetected"
	InstrumentationInProgress State = "InstrumentationInProgress"
	InstrumentedState         State = "Instrumented"
)

type Orchestrator struct {
	Client          *kube.Client
	OdigosNamespace string
	TransitionsMap  map[string]Transition
}

func NewOrchestrator(client *kube.Client, ctx context.Context) (*Orchestrator, error) {
	ns, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return nil, err
	}

	transitions := make(map[string]Transition)
	for _, t := range allTransitions {
		t.Init(client)
		transitions[string(t.From())] = t
	}

	return &Orchestrator{Client: client,
		OdigosNamespace: ns,
		TransitionsMap:  transitions,
	}, nil
}

func (o *Orchestrator) Apply(ctx context.Context, obj client.Object, templateSpec *v1.PodTemplateSpec) {
	state := o.getCurrentState(ctx, obj, templateSpec)
	o.log(fmt.Sprintf("Current state: %s", state))
	nextTransition := o.TransitionsMap[string(state)]
	for nextTransition != nil {
		err := nextTransition.Execute(ctx, obj, templateSpec)
		if err != nil {
			o.log(fmt.Sprintf("Error executing transition: %s", err))
			return
		}

		// Special case: PreflightCheck change state manualy
		if nextTransition.To() == PreflightChecksPassed {
			state = PreflightChecksPassed
		} else {
			state = o.getCurrentState(ctx, obj, templateSpec)
		}
		o.log(fmt.Sprintf("Current state: %s", state))
		nextTransition = o.TransitionsMap[string(state)]
	}
}

func (o *Orchestrator) getCurrentState(ctx context.Context, obj client.Object, templateSpec *v1.PodTemplateSpec) State {
	if !workload.IsObjectLabeledForInstrumentation(obj) {
		return NotInstrumentedState
	}

	name := obj.GetName()
	kind := workload.WorkloadKindFromClientObject(obj)
	icName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	_, err := o.Client.OdigosClient.InstrumentationConfigs(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return LangDetectionInProgress
		}

		o.log(fmt.Sprintf("Error getting instrumentation config: %s, skipping", err))
		return UnknownState
	}

	return UnknownState
}

func (o *Orchestrator) log(str string) {
	fmt.Printf("    > %s\n", str)
}
