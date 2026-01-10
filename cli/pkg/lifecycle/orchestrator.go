package lifecycle

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	openshiftappsv1 "github.com/openshift/api/apps/v1"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type State int

const (
	// Unknown is an error state.
	StateUnknown State = iota

	// NoSourceCreated is the initial state before instrumentation starts.
	StateNoSourceCreated

	// PreflightChecksPassed indicates the workload is healthy and can be instrumented.
	StatePreflightChecksPassed

	// SourceCreated indicates the Source object has been created for the workload and language detection is in progress.
	StateSourceCreated

	// LangDetected indicates the language detection has completed and the workload is eligible for instrumentation.
	StateLangDetected

	// InstrumentationInProgress indicates the instrumentation is in progress, but the pods are not yet running.
	StateInstrumentationInProgress

	// Instrumented indicates the instrumentation has completed and the workload is instrumented.
	StateInstrumented

	// PostCheckPassed indicates all pods in the workload are healthy and the post check has completed and the workload is ready for use.
	StatePostCheckPassed
)

func (s State) String() string {
	switch s {
	case StateUnknown:
		return "Unknown"
	case StateNoSourceCreated:
		return "NoSourceCreated"
	case StatePreflightChecksPassed:
		return "PreflightChecksPassed"
	case StateSourceCreated:
		return "SourceCreated"
	case StateLangDetected:
		return "LangDetected"
	case StateInstrumentationInProgress:
		return "InstrumentationInProgress"
	case StateInstrumented:
		return "Instrumented"
	case StatePostCheckPassed:
		return "PostCheckPassed"
	default:
		return "Unknown"
	}
}

type Orchestrator struct {
	Client          *kube.Client
	OdigosNamespace string
	TransitionsMap  map[State]Transition
	Remote          bool
}

func NewOrchestrator(client *kube.Client, ctx context.Context, isRemote bool) (*Orchestrator, error) {
	ns, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return nil, err
	}

	baseTransition := BaseTransition{client: client, odigosNamespace: ns, remote: isRemote}

	// stateToTransitionMap is a map of states to transitions.
	// The current state determines which transition to execute while an app is in that state.
	var stateToTransitionMap = map[State]Transition{
		// When an app is not currently instrumented, we need to check if it's eligible for instrumentation.
		StateNoSourceCreated: &PreflightCheck{baseTransition},

		// If the app is eligible for instrumentation, we need to request the language detection.
		StatePreflightChecksPassed: &RequestLangDetection{baseTransition},

		// If the language detection is in progress, we need to wait for it to complete.
		StateSourceCreated: &WaitForLangDetection{baseTransition},

		// If the language detection is complete, we need to start the instrumentation.
		StateLangDetected: &InstrumentationStarted{baseTransition},

		// If the instrumentation is in progress, we need to wait for it to complete.
		StateInstrumentationInProgress: &InstrumentationEnded{baseTransition},

		// If the instrumentation is complete, check that all pods are healthy.
		StateInstrumented: &PostCheck{baseTransition},

		// If the post check is complete, the workload is ready for use.
		StatePostCheckPassed: nil,
	}

	return &Orchestrator{Client: client,
		OdigosNamespace: ns,
		TransitionsMap:  stateToTransitionMap,
		Remote:          isRemote,
	}, nil
}

// Apply is called sequentially on workloads to instrument them.
func (o *Orchestrator) Apply(ctx context.Context, obj metav1.Object) error {
	// Create a channel to handle cancellation
	done := make(chan struct{})
	var finalErr error

	go func() {
		defer close(done)

		// Start by assuming the app is not instrumented and run preflight checks.
		state := StateNoSourceCreated
		prevState := StateUnknown
		currentTransition := o.TransitionsMap[state]

		for currentTransition != nil {
			select {
			case <-ctx.Done():
				finalErr = ctx.Err()
				return
			default:

				// Execute the current transition
				if err := currentTransition.Execute(ctx, obj); err != nil {
					o.log(fmt.Sprintf("Error executing transition: %s", err))
					finalErr = fmt.Errorf("failed to execute transition: %w", err)
					return
				}

				currentState, err := o.getCurrentState(ctx, obj)
				if err != nil {
					o.log(fmt.Sprintf("Error getting current state: %s", err))
					finalErr = fmt.Errorf("failed to get current state: %w", err)
					return
				}
				state = currentState
				if state != prevState {
					o.log(fmt.Sprintf("Current state: %s", state.String()))
					prevState = state
				}

				if state == StateUnknown {
					o.log(fmt.Sprintf("Unknown state: %s", state.String()))
					finalErr = fmt.Errorf("unknown state: %s", state.String())
					return
				}

				currentTransition = o.TransitionsMap[state]
			}
		}
	}()

	// Wait for either completion or context cancellation
	select {
	case <-ctx.Done():
		// Wait for the goroutine to finish rollback
		<-done
		if finalErr == nil {
			return ctx.Err()
		}
		return finalErr
	case <-done:
		return finalErr
	}
}

func (o *Orchestrator) getCurrentState(ctx context.Context, obj metav1.Object) (State, error) {
	for state := StateNoSourceCreated; state <= StatePostCheckPassed; state++ {
		transition := o.TransitionsMap[state]
		if transition == nil {
			// PostCheckPassed is the last state, so if we reach it, we can return it.
			break
		}
		transitionState, err := transition.GetTransitionState(ctx, obj)
		if err != nil || transitionState == StateUnknown {
			return StateUnknown, err
		}
		// If the transition returns its From() state (the state we came from), then we need to run the transition for that state.
		if transitionState == state {
			return state, nil
		}
	}
	return StatePostCheckPassed, nil
}

func (o *Orchestrator) log(str string) {
	fmt.Printf("    > %s\n", str)
}

type Transition interface {
	From() State
	To() State
	Execute(ctx context.Context, obj metav1.Object) error
	GetTransitionState(ctx context.Context, obj metav1.Object) (State, error)
}

type BaseTransition struct {
	client          *kube.Client
	odigosNamespace string
	remote          bool
}

func (b *BaseTransition) log(str string) {
	fmt.Printf("    > %s\n", str)
}

func WorkloadKindFrombject(obj metav1.Object) k8sconsts.WorkloadKind {
	switch obj.(type) {
	case *v1.Deployment:
		return k8sconsts.WorkloadKindDeployment
	case *v1.StatefulSet:
		return k8sconsts.WorkloadKindStatefulSet
	case *v1.DaemonSet:
		return k8sconsts.WorkloadKindDaemonSet
	case *batchv1beta1.CronJob:
		return k8sconsts.WorkloadKindCronJob
	case *batchv1.CronJob:
		return k8sconsts.WorkloadKindCronJob
	case *openshiftappsv1.DeploymentConfig:
		return k8sconsts.WorkloadKindDeploymentConfig
	case *argorolloutsv1alpha1.Rollout:
		return k8sconsts.WorkloadKindArgoRollout
	default:
		return k8sconsts.WorkloadKind("")
	}
}
