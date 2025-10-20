package lifecycle

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PreflightCheck struct {
	BaseTransition
}

func (p *PreflightCheck) From() State {
	return StateNoSourceCreated
}

func (p *PreflightCheck) To() State {
	return StatePreflightChecksPassed
}

func (p *PreflightCheck) Execute(ctx context.Context, obj metav1.Object) error {
	allPodsRunning, err := utils.VerifyAllPodsAreRunning(ctx, p.client, obj, false)
	if err != nil {
		return err
	}
	if !allPodsRunning {
		return errors.New("Workload is not ready")
	}

	return nil
}

func (p *PreflightCheck) GetTransitionState(ctx context.Context, obj metav1.Object) (State, error) {
	// If Execute passed, then the transition is successful.
	return p.To(), nil
}

var _ Transition = &PreflightCheck{}
