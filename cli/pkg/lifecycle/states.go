package lifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"

	"github.com/odigos-io/odigos/cli/pkg/remote"

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"github.com/odigos-io/odigos/common"

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

type PodTemplateSpecFetcher func(ctx context.Context, name string, namespace string) (*v1.PodTemplateSpec, error)

type Orchestrator struct {
	Client          *kube.Client
	OdigosNamespace string
	TransitionsMap  map[string]Transition
	Remote          bool
}

func NewOrchestrator(client *kube.Client, ctx context.Context, isRemote bool) (*Orchestrator, error) {
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
		Remote:          isRemote,
	}, nil
}

func (o *Orchestrator) Apply(ctx context.Context, obj client.Object, templateSpecFetcher PodTemplateSpecFetcher) error {
	// Create a channel to handle cancellation
	done := make(chan struct{})
	var finalErr error

	go func() {
		defer close(done)

		templateSpec, err := templateSpecFetcher(ctx, obj.GetName(), obj.GetNamespace())
		if err != nil {
			o.log(fmt.Sprintf("Error fetching pod template spec: %s", err))
			finalErr = fmt.Errorf("failed to fetch template spec: %w", err)
			return
		}

		state := o.getCurrentState(ctx, obj, templateSpec)
		o.log(fmt.Sprintf("Current state: %s", state))

		if state == UnknownState {
			if err := o.rollBack(obj, templateSpecFetcher); err != nil {
				o.log(fmt.Sprintf("Error rolling back: %s", err))
				finalErr = fmt.Errorf("failed to rollback from unknown state: %w", err)
				return
			}
			return
		}

		nextTransition := o.TransitionsMap[string(state)]
		for nextTransition != nil {
			select {
			case <-ctx.Done():
				// Context was cancelled, perform rollback
				o.log("Context cancelled, rolling back current object")
				if err := o.rollBack(obj, templateSpecFetcher); err != nil {
					o.log(fmt.Sprintf("Error rolling back after context cancellation: %s", err))
					finalErr = fmt.Errorf("failed to rollback after context cancellation: %w", err)
					return
				}
				finalErr = ctx.Err()
				return
			default:
				templateSpec, err := templateSpecFetcher(ctx, obj.GetName(), obj.GetNamespace())
				if err != nil {
					o.log(fmt.Sprintf("Error fetching pod template spec: %s", err))
					finalErr = fmt.Errorf("failed to fetch template spec during transition: %w", err)
					return
				}

				if err := nextTransition.Execute(ctx, obj, templateSpec, o.Remote); err != nil {
					o.log(fmt.Sprintf("Error executing transition: %s", err))
					// Attempt rollback on execution error
					if rbErr := o.rollBack(obj, templateSpecFetcher); rbErr != nil {
						o.log(fmt.Sprintf("Error rolling back after failed execution: %s", rbErr))
						finalErr = fmt.Errorf("failed to rollback after execution error: %w", rbErr)
						return
					}
					finalErr = fmt.Errorf("failed to execute transition: %w", err)
					return
				}

				// Special case: PreflightCheck change state manually
				if nextTransition.To() == PreflightChecksPassed {
					state = PreflightChecksPassed
				} else {
					state = o.getCurrentState(ctx, obj, templateSpec)
				}

				o.log(fmt.Sprintf("Current state: %s", state))

				if state == UnknownState {
					if err := o.rollBack(obj, templateSpecFetcher); err != nil {
						o.log(fmt.Sprintf("Error rolling back: %s", err))
						finalErr = fmt.Errorf("failed to rollback from unknown state during transition: %w", err)
						return
					}
					return
				}

				nextTransition = o.TransitionsMap[string(state)]
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

func (o *Orchestrator) getCurrentState(ctx context.Context, obj client.Object, templateSpec *v1.PodTemplateSpec) State {
	if !workload.IsObjectLabeledForInstrumentation(obj) {
		return NotInstrumentedState
	}

	name := obj.GetName()
	kind := workload.WorkloadKindFromClientObject(obj)
	icName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	var describe *source.SourceAnalyze
	var err error
	if o.Remote {
		describe, err = remote.DescribeSource(ctx, o.Client, o.OdigosNamespace, string(kind), obj.GetNamespace(), name)
		if err != nil {
			o.log(fmt.Sprintf("Error describing source: %s", err))
			return UnknownState
		}

		if describe == nil {
			o.log("Describe source returned nil, skipping")
			return UnknownState
		}
	}

	if !o.Remote {
		_, err := o.Client.OdigosClient.InstrumentationConfigs(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return LangDetectionInProgress
			}

			o.log(fmt.Sprintf("Error getting instrumentation config: %s, skipping", err))
			return UnknownState
		}
	} else {
		if describe.InstrumentationConfig.Created.Value == nil {
			return LangDetectionInProgress
		}

		instConfigStr, ok := describe.InstrumentationConfig.Created.Value.(string)
		if !ok {
			o.log("Failed to get instrumentation config status, skipping")
			return UnknownState
		}

		if instConfigStr != "created" {
			return LangDetectionInProgress
		}
	}

	var lang common.ProgrammingLanguage
	if !o.Remote {
		ia, err := o.Client.OdigosClient.InstrumentedApplications(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return LangDetectionInProgress
			}

			o.log(fmt.Sprintf("Error getting instrumented application: %s, skipping", err))
			return UnknownState
		}

		if ia.Spec.RuntimeDetails == nil || len(ia.Spec.RuntimeDetails) == 0 {
			return LangDetectionInProgress
		}

		langFound := false
		for _, rd := range ia.Spec.RuntimeDetails {
			if rd.Language != common.UnknownProgrammingLanguage && rd.Language != common.IgnoredProgrammingLanguage {
				langFound = true
				lang = rd.Language
				break
			}
		}

		if !langFound {
			o.log("Failed to deetect language, skipping")
			return UnknownState
		}
	} else {
		if describe.InstrumentedApplication.Created.Value == nil {
			return LangDetectionInProgress
		}

		iaCreated, ok := describe.InstrumentedApplication.Created.Value.(string)
		if !ok {
			o.log("Failed to get instrumented application status, skipping")
			return UnknownState
		}

		if iaCreated != "created" {
			return LangDetectionInProgress
		}

		if len(describe.InstrumentedApplication.Containers) == 0 {
			return LangDetectionInProgress
		}

		langFound := false
		for _, c := range describe.InstrumentedApplication.Containers {
			langStr, ok := c.Language.Value.(string)
			if !ok {
				continue
			}

			langParsed := common.ProgrammingLanguage(langStr)
			if langParsed != common.UnknownProgrammingLanguage && langParsed != common.IgnoredProgrammingLanguage {
				langFound = true
				lang = langParsed
				break
			}
		}

		if !langFound {
			o.log("Failed to deetect language, skipping")
			return UnknownState
		}
	}

	instDeviceFound := false
	for _, c := range templateSpec.Spec.Containers {
		if c.Resources.Limits != nil {
			for val := range c.Resources.Limits {
				if strings.HasPrefix(val.String(), common.OdigosResourceNamespace) {
					instDeviceFound = true
					break
				}
			}
		}
	}

	if !instDeviceFound {
		o.log("Language detected: " + string(lang))
		return LangDetectedState
	}

	instrumented, err := k8sutils.VerifyAllPodsAreInstrumented(ctx, o.Client, obj)
	if err != nil {
		o.log(fmt.Sprintf("Error verifying all pods are instrumented: %s", err))
		return UnknownState
	}

	if !instrumented {
		return InstrumentationInProgress
	}

	// TODO(edenfed): If relevant language + InstrumentationInstance does not exists = InstrumentationInProgress

	return InstrumentedState
}

func (o *Orchestrator) log(str string) {
	fmt.Printf("    > %s\n", str)
}
