package lifecycle

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/remote"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (o *Orchestrator) Apply(ctx context.Context, obj client.Object) error {
	// Create a channel to handle cancellation
	done := make(chan struct{})
	var finalErr error

	go func() {
		defer close(done)
		state := o.getCurrentState(ctx, obj)
		o.log(fmt.Sprintf("Current state: %s", state))

		nextTransition := o.TransitionsMap[string(state)]
		for nextTransition != nil {
			select {
			case <-ctx.Done():
				finalErr = ctx.Err()
				return
			default:
				if err := nextTransition.Execute(ctx, obj, o.Remote); err != nil {
					o.log(fmt.Sprintf("Error executing transition: %s", err))
					finalErr = fmt.Errorf("failed to execute transition: %w", err)
					return
				}

				// Special case: PreflightCheck change state manually
				if nextTransition.To() == PreflightChecksPassed {
					state = PreflightChecksPassed
				} else {
					state = o.getCurrentState(ctx, obj)
				}
				o.log(fmt.Sprintf("Current state: %s", state))

				if state == UnknownState {
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

func (o *Orchestrator) getCurrentState(ctx context.Context, obj client.Object) State {
	var kind k8sconsts.WorkloadKind
	switch obj.(type) {
	case *appsv1.Deployment:
		kind = k8sconsts.WorkloadKindDeployment
	case *appsv1.StatefulSet:
		kind = k8sconsts.WorkloadKindStatefulSet
	case *appsv1.DaemonSet:
		kind = k8sconsts.WorkloadKindDaemonSet
	case *batchv1beta1.CronJob:
		kind = k8sconsts.WorkloadKindCronJob
	case *batchv1.CronJob:
		kind = k8sconsts.WorkloadKindCronJob
	}

	var describe *source.SourceAnalyze
	if !o.Remote {
		labeled := labels.Set{
			k8sconsts.WorkloadNameLabel:      obj.GetName(),
			k8sconsts.WorkloadNamespaceLabel: obj.GetNamespace(),
			k8sconsts.WorkloadKindLabel:      string(kind),
		}
		sources, err := o.Client.OdigosClient.Sources(obj.GetNamespace()).List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labeled).String()})
		if err != nil {
			return UnknownState
		}
		if len(sources.Items) == 0 {
			return NotInstrumentedState
		}
	} else {
		des, err := remote.DescribeSource(ctx, o.Client, o.OdigosNamespace, string(kind), obj.GetNamespace(), obj.GetName())
		if err != nil || des.Name.Value == nil {
			// name value will be nil for unsupported kinds
			return UnknownState
		}

		if des.SourceObjectsAnalysis.Instrumented.Value != true {
			return NotInstrumentedState
		}
		describe = des
	}

	icName := workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), kind)

	// Check if the instrumentation config exists for lang detection
	var ic *odigosv1.InstrumentationConfig
	var lang common.ProgrammingLanguage
	if !o.Remote {
		instrumentationConfig, err := o.Client.OdigosClient.InstrumentationConfigs(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return LangDetectionInProgress
			}
			return UnknownState
		}
		ic = instrumentationConfig
		if ic.Status.Conditions == nil {
			if status := meta.FindStatusCondition(ic.Status.Conditions, odigosv1.RuntimeDetectionStatusConditionType); status != nil {
				if status.Reason == string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully) && status.Status != metav1.ConditionTrue {
					return LangDetectionInProgress
				}
			}
		}

		if ic.Spec.SdkConfigs == nil || len(ic.Spec.SdkConfigs) == 0 {
			return LangDetectionInProgress
		}

		langFound := false
		for _, sdkConfig := range ic.Spec.SdkConfigs {
			if sdkConfig.Language != common.UnknownProgrammingLanguage && sdkConfig.Language != common.IgnoredProgrammingLanguage {
				lang = sdkConfig.Language
				langFound = true
				break
			}
		}

		if !langFound {
			o.log("Failed to detect language, skipping")
			return UnknownState
		}
	} else {
		if describe.RuntimeInfo == nil {
			return LangDetectionInProgress
		}

		if len(describe.RuntimeInfo.Containers) == 0 {
			return LangDetectionInProgress
		}

		langFound := false
		for _, c := range describe.RuntimeInfo.Containers {
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
			o.log("Failed to detect language, skipping")
			return UnknownState
		}
	}

	o.log(fmt.Sprintf("Language detected: %s", lang))

	if kind != k8sconsts.WorkloadKindCronJob && kind != k8sconsts.WorkloadKindJob {
		instrumented, err := utils.VerifyAllPodsAreRunning(ctx, o.Client, obj)
		if err != nil {
			o.log(fmt.Sprintf("Error verifying all pods are instrumented: %s", err))
			return UnknownState
		}

		if !instrumented {
			return InstrumentationInProgress
		}
	}

	return InstrumentedState
}

func (o *Orchestrator) log(str string) {
	fmt.Printf("    > %s\n", str)
}

type Transition interface {
	From() State
	To() State
	Execute(ctx context.Context, obj client.Object, isRemote bool) error
	Init(client *kube.Client)
}

type BaseTransition struct {
	client *kube.Client
}

func (b *BaseTransition) Init(client *kube.Client) {
	b.client = client
}

func (b *BaseTransition) log(str string) {
	fmt.Printf("    > %s\n", str)
}

var allTransitions = []Transition{
	&PreflightCheck{},
	&RequestLangDetection{},
	&WaitForLangDetection{},
	&InstrumentationStarted{},
	&InstrumentationEnded{},
}
