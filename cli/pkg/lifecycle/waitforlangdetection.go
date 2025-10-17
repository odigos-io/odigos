package lifecycle

import (
	"context"
	"errors"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/pkg/remote"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WaitForLangDetection struct {
	BaseTransition
}

func (w *WaitForLangDetection) From() State {
	return LangDetectionInProgress
}

func (w *WaitForLangDetection) To() State {
	return LangDetectedState
}

func (w *WaitForLangDetection) Execute(ctx context.Context, obj client.Object, isRemote bool) error {
	workloadKind := workload.WorkloadKindFromClientObject(obj)
	icName := workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), workloadKind)

	return wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		if !isRemote {
			ic, err := w.client.OdigosClient.InstrumentationConfigs(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
			if err != nil {
				if !apierrors.IsNotFound(err) {
					w.log("Error while fetching InstrumentationConfig: " + err.Error())
				}
				return false, nil
			}

			if status := meta.FindStatusCondition(ic.Status.Conditions, odigosv1.RuntimeDetectionStatusConditionType); status != nil {
				if status.Reason == string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully) && status.Status == metav1.ConditionTrue {
					return true, nil
				}
				return false, nil
			}
		}

		describe, err := remote.DescribeSource(ctx, w.client, obj.GetNamespace(), string(workloadKind), obj.GetNamespace(), obj.GetName())
		if err != nil {
			return false, nil
		}

		if describe.RuntimeInfo == nil {
			return false, nil
		}

		if len(describe.RuntimeInfo.Containers) == 0 {
			return false, nil
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
				break
			}
		}
		if !langFound {
			return false, errors.New("Failed to detect language")
		}

		return true, nil
	})
}

func (w *WaitForLangDetection) GetTransitionState(ctx context.Context, obj client.Object, isRemote bool, odigosNamespace string) (State, error) {
	workloadKind := workload.WorkloadKindFromClientObject(obj)
	icName := workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), workloadKind)

	if !isRemote {
		instrumentationConfig, err := w.client.OdigosClient.InstrumentationConfigs(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return w.From(), nil
			}
			return UnknownState, err
		}
		if instrumentationConfig.Status.Conditions == nil {
			if status := meta.FindStatusCondition(instrumentationConfig.Status.Conditions, odigosv1.RuntimeDetectionStatusConditionType); status != nil {
				if status.Reason == string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully) && status.Status != metav1.ConditionTrue {
					return w.From(), nil
				}
			}
		}

		if instrumentationConfig.Spec.SdkConfigs == nil || len(instrumentationConfig.Spec.SdkConfigs) == 0 {
			return w.From(), nil
		}

		langFound := false
		for _, sdkConfig := range instrumentationConfig.Spec.SdkConfigs {
			if sdkConfig.Language != common.UnknownProgrammingLanguage && sdkConfig.Language != common.IgnoredProgrammingLanguage {
				langFound = true
				break
			}
		}

		if !langFound {
			w.log("Failed to detect language, skipping")
			return UnknownState, nil
		}
	} else {
		describe, err := remote.DescribeSource(ctx, w.client, odigosNamespace, string(workloadKind), obj.GetNamespace(), obj.GetName())
		if err != nil {
			return UnknownState, err
		}
		if describe.RuntimeInfo == nil {
			return w.From(), nil
		}

		if len(describe.RuntimeInfo.Containers) == 0 {
			return w.From(), nil
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
				break
			}
		}
		if !langFound {
			w.log("Failed to detect language, skipping")
			return UnknownState, nil
		}
	}

	return w.To(), nil
}

var _ Transition = &WaitForLangDetection{}
