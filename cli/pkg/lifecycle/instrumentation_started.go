package lifecycle

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationStarted struct {
	BaseTransition
}

func (i *InstrumentationStarted) From() State {
	return LangDetectedState
}

func (i *InstrumentationStarted) To() State {
	return InstrumentationInProgress
}

func (i *InstrumentationStarted) Execute(ctx context.Context, obj client.Object, isRemote bool) error {
	return nil
}

var _ Transition = &InstrumentationStarted{}
