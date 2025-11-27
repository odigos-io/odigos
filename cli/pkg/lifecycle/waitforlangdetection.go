package lifecycle

import (
	"context"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/pkg/remote"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type WaitForLangDetection struct {
	BaseTransition
}

func (w *WaitForLangDetection) From() State {
	return StateSourceCreated
}

func (w *WaitForLangDetection) To() State {
	return StateLangDetected
}

// checkLanguageDetected checks if language detection has completed for the workload.
// Returns (detected, error) where:
// - detected is true if language detection succeeded
// - error is non-nil only for actual errors (not for "not ready yet" cases)
func (w *WaitForLangDetection) checkLanguageDetected(ctx context.Context, obj metav1.Object) (bool, error) {
	workloadKind := WorkloadKindFrombject(obj)
	icName := workload.CalculateWorkloadRuntimeObjectName(obj.GetName(), workloadKind)

	if !w.remote {
		ic, err := w.client.OdigosClient.InstrumentationConfigs(obj.GetNamespace()).Get(ctx, icName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		// Check status conditions
		if status := meta.FindStatusCondition(ic.Status.Conditions, odigosv1.RuntimeDetectionStatusConditionType); status != nil {
			if status.Reason == string(odigosv1.RuntimeDetectionReasonDetectedSuccessfully) && status.Status == metav1.ConditionTrue {
				// Condition indicates success, now verify SdkConfigs
				if ic.Spec.SdkConfigs == nil || len(ic.Spec.SdkConfigs) == 0 {
					return false, nil
				}

				// Check if at least one valid language was detected
				for _, sdkConfig := range ic.Spec.SdkConfigs {
					if sdkConfig.Language != common.UnknownProgrammingLanguage {
						return true, nil
					}
				}
				return false, nil
			}
		}
	} else {
		// Remote mode: use DescribeSource API
		describe, err := remote.DescribeSource(ctx, w.client, w.odigosNamespace, string(workloadKind), obj.GetNamespace(), obj.GetName())
		if err != nil {
			return false, nil
		}

		if describe.RuntimeInfo == nil || len(describe.RuntimeInfo.Containers) == 0 {
			return false, nil
		}

		// Check if at least one container has a detected language
		for _, c := range describe.RuntimeInfo.Containers {
			langStr, ok := c.Language.Value.(string)
			if !ok {
				continue
			}

			langParsed := common.ProgrammingLanguage(langStr)
			if langParsed != common.UnknownProgrammingLanguage {
				return true, nil
			}
		}
	}

	return false, nil
}

func (w *WaitForLangDetection) Execute(ctx context.Context, obj metav1.Object) error {
	return wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		detected, err := w.checkLanguageDetected(ctx, obj)
		if err != nil {
			w.log("Error checking language detection: " + err.Error())
			return false, nil
		}
		return detected, nil
	})
}

func (w *WaitForLangDetection) GetTransitionState(ctx context.Context, obj metav1.Object) (State, error) {
	detected, err := w.checkLanguageDetected(ctx, obj)
	if err != nil {
		return StateUnknown, err
	}

	if !detected {
		return w.From(), nil
	}

	return w.To(), nil
}

var _ Transition = &WaitForLangDetection{}
