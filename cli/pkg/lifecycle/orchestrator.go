package lifecycle

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type State string

const (
	// UnknownState indicates something went wrong and we don't know the current state.
	UnknownState State = "Unknown"

	// NoSourceCreatedState indicates that no Source object has been created for the workload.
	NoSourceCreatedState State = "NoSourceCreated"

	// PreflightChecksPassedState indicates that the preflight checks have passed and the workload is eligible for instrumentation.
	PreflightChecksPassed State = "PreflightChecksPassed"

	// SourceCreatedState indicates that the Source object has been created for the workload and language detection is in progress.
	SourceCreated State = "SourceCreated"

	// LangDetectedState indicates that the language detection has completed and the workload is eligible for instrumentation.
	LangDetectedState State = "LangDetected"

	// InstrumentationInProgressState indicates that the instrumentation is in progress, but the pods are not yet running.
	InstrumentationInProgress State = "InstrumentationInProgress"

	// InstrumentedState indicates that the instrumentation has completed and the workload is instrumented and the pods are running.
	InstrumentedState State = "Instrumented"
)

var orderedStates = []State{
	NoSourceCreatedState,
	PreflightChecksPassed,
	SourceCreated,
	LangDetectedState,
	InstrumentationInProgress,
	InstrumentedState,
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

	baseTransition := BaseTransition{client: client}

	// stateToTransitionMap is a map of states to transitions.
	// The current state determines which transition to execute while an app is in that state.
	var stateToTransitionMap = map[State]Transition{
		// When an app is not currently instrumented, we need to check if it's eligible for instrumentation.
		NoSourceCreatedState: &PreflightCheck{baseTransition},

		// If the app is eligible for instrumentation, we need to request the language detection.
		PreflightChecksPassed: &RequestLangDetection{baseTransition},

		// If the language detection is in progress, we need to wait for it to complete.
		SourceCreated: &WaitForLangDetection{baseTransition},

		// If the language detection is complete, we need to start the instrumentation.
		LangDetectedState: &InstrumentationStarted{baseTransition},

		// If the instrumentation is in progress, we need to wait for it to complete.
		InstrumentationInProgress: &InstrumentationEnded{baseTransition},

		// If the instrumentation is complete, we don't need to do anything.
		InstrumentedState: nil,
	}

	return &Orchestrator{Client: client,
		OdigosNamespace: ns,
		TransitionsMap:  stateToTransitionMap,
		Remote:          isRemote,
	}, nil
}

func (o *Orchestrator) Apply(ctx context.Context, obj client.Object) error {
	// Create a channel to handle cancellation
	done := make(chan struct{})
	var finalErr error

	go func() {
		defer close(done)

		// Start by assuming the app is not instrumented and run preflight checks.
		state := NoSourceCreatedState
		prevState := UnknownState
		currentTransition := o.TransitionsMap[state]

		for currentTransition != nil {
			select {
			case <-ctx.Done():
				finalErr = ctx.Err()
				return
			default:

				// Execute the current transition
				if err := currentTransition.Execute(ctx, obj, o.Remote); err != nil {
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
					o.log(fmt.Sprintf("Current state: %s", state))
					prevState = state
				}

				if state == UnknownState {
					o.log(fmt.Sprintf("Unknown state: %s", state))
					finalErr = fmt.Errorf("unknown state: %s", state)
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

func (o *Orchestrator) getCurrentState(ctx context.Context, obj client.Object) (State, error) {
	for _, state := range orderedStates {
		transition := o.TransitionsMap[state]
		if transition == nil {
			// InstrumentedState is the last state, so if we reach it, we can return it.
			break
		}
		transitionState, err := transition.GetTransitionState(ctx, obj, o.Remote, o.OdigosNamespace)
		if err != nil || transitionState == UnknownState {
			return UnknownState, err
		}
		// If the transition returns its From() state (the state we came from), then we need to run the transition for that state.
		if transitionState == state {
			return state, nil
		}
	}
	return InstrumentedState, nil
}

func (o *Orchestrator) log(str string) {
	fmt.Printf("    > %s\n", str)
}

type Transition interface {
	From() State
	To() State
	Execute(ctx context.Context, obj client.Object, isRemote bool) error
	GetTransitionState(ctx context.Context, obj client.Object, isRemote bool, odigosNamespace string) (State, error)
}

type BaseTransition struct {
	client *kube.Client
}

func (b *BaseTransition) log(str string) {
	fmt.Printf("    > %s\n", str)
}
