package lifecycle

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PreflightCheck struct {
	BaseTransition
}

func (p *PreflightCheck) From() State {
	return NoSourceCreatedState
}

func (p *PreflightCheck) To() State {
	return PreflightChecksPassed
}

func (p *PreflightCheck) Execute(ctx context.Context, obj client.Object) error {
	allPodsRunning, err := utils.VerifyAllPodsAreRunning(ctx, p.client, obj)
	if err != nil {
		return err
	}
	if !allPodsRunning {
		return errors.New("Workload is not ready")
	}

	return nil
}

func (p *PreflightCheck) GetTransitionState(ctx context.Context, obj client.Object) (State, error) {
	// If Execute passed, then the transition is successful.
	return p.To(), nil
}

var _ Transition = &PreflightCheck{}
