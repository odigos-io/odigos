package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

type ActionReconciler struct {
	client.Client
}

type ActionConfig interface {
	ProcessorType() string
	OrderHint() int
	CollectorRoles() []k8sconsts.CollectorRole
}

// giving an action, return it's specific processor details
// returns the type of the processors, order hint, config, and error
func convertActionToProcessor(ctx context.Context, k8sclient client.Client, action *odigosv1.Action) (*odigosv1.Processor, error) {
	var config any
	if action.Spec.AddClusterInfo != nil {
		config = addClusterInfoConfig(action.Spec.AddClusterInfo.ClusterAttributes, action.Spec.AddClusterInfo.OverwriteExistingValues)
		return convertToDefaultProcessor(action, action.Spec.AddClusterInfo, config)
	}

	if action.Spec.DeleteAttribute != nil {
		config, err := deleteAttributeConfig(action.Spec.DeleteAttribute.AttributeNamesToDelete, action.Spec.Signals)
		if err != nil {
			return nil, err
		}
		return convertToDefaultProcessor(action, action.Spec.DeleteAttribute, config)
	}

	if action.Spec.PiiMasking != nil {
		for _, signal := range action.Spec.Signals {
			if _, ok := piiMaskingSupportedSignals[signal]; !ok {
				return nil, fmt.Errorf("unsupported signal in PiiMasking action: %s", signal)
			}
		}
		config, err := piiMaskingConfig(action.Spec.PiiMasking.PiiCategories)
		if err != nil {
			return nil, err
		}
		return convertToDefaultProcessor(action, action.Spec.PiiMasking, config)
	}

	if action.Spec.RenameAttribute != nil {
		config, err := renameAttributeConfig(action.Spec.RenameAttribute.Renames, action.Spec.Signals)
		if err != nil {
			return nil, err
		}
		return convertToDefaultProcessor(action, action.Spec.RenameAttribute, config)
	}

	if action.Spec.K8sAttributes != nil {
		config, signals, ownerReferences, err := k8sAttributeConfig(ctx, k8sclient, action.Namespace)
		if err != nil {
			return nil, err
		}
		configJson, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}
		processor := &odigosv1.Processor{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "odigos.io/v1alpha1",
				Kind:       "Processor",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "odigos-k8sattributes",
				Namespace:       action.Namespace,
				OwnerReferences: ownerReferences,
			},
			Spec: odigosv1.ProcessorSpec{
				Type:            action.Spec.K8sAttributes.ProcessorType(),
				ProcessorName:   "Unified Kubernetes Attributes",
				Disabled:        false,
				Notes:           action.Spec.Notes,
				CollectorRoles:  []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleNodeCollector},
				OrderHint:       action.Spec.K8sAttributes.OrderHint(),
				ProcessorConfig: runtime.RawExtension{Raw: configJson},
			},
		}
		for signal := range signals {
			processor.Spec.Signals = append(processor.Spec.Signals, signal)
		}
		return processor, nil
	}

	if action.Spec.Samplers != nil {
		// Handle probabilistic sampler separately since it has different processor requirements
		if action.Spec.Samplers.ProbabilisticSampler != nil {
			for _, signal := range action.Spec.Signals {
				if _, ok := supportedProbabilisticSignals[signal]; !ok {
					return nil, fmt.Errorf("unsupported signal: %s", signal)
				}
			}

			// Convert string percentage to float
			config, err := probabilisticSamplerConfig(action.Spec.Samplers.ProbabilisticSampler.SamplingPercentage)
			if err != nil {
				return nil, err
			}

			// Create probabilistic sampler processor using its own config
			return convertToDefaultProcessor(action, action.Spec.Samplers.ProbabilisticSampler, config)
		} else {
			// Handle other samplers using the unified sampling approach
			// For non-probabilistic samplers, we need to create a composite config
			// that can be used with the odigossampling processor
			config, ownerReferences, err := samplersConfig(ctx, k8sclient, action.Namespace)
			if err != nil {
				return nil, err
			}
			if config == nil {
				return nil, nil
			}

			samplingConfigJson, err := json.Marshal(config)
			if err != nil {
				return nil, err
			}

			groupByTraceProcessor := getGroupByTraceProcessor(action.Namespace, ownerReferences)
			if err := k8sclient.Patch(ctx, groupByTraceProcessor, client.Apply, client.FieldOwner("groupbytrace"), client.ForceOwnership); err != nil {
				return nil, err
			}

			processor := &v1.Processor{
				TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "sampling-processor",
					Namespace:       action.Namespace,
					OwnerReferences: ownerReferences,
				},
				Spec: v1.ProcessorSpec{
					Type:            action.Spec.Samplers.ProcessorType(),
					ProcessorName:   action.Spec.Samplers.ProcessorType(),
					Disabled:        false, // In case related actions are disabled, the processor won't be created
					Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
					CollectorRoles:  []v1.CollectorsGroupRole{v1.CollectorsGroupRoleClusterGateway},
					OrderHint:       action.Spec.Samplers.OrderHint(),
					ProcessorConfig: runtime.RawExtension{Raw: samplingConfigJson},
				},
			}

			return processor, nil
		}
	}

	return nil, errors.New("no supported action found in resource")
}

// returns a processor object with:
// - ns and name similar to the action name
// - signals based on the action signals
// - owner reference to the action
// - type and order hint based on the function input
// - config based on the function input, stringified in JSON
// - collector roles set to ClusterGateway
func convertToDefaultProcessor(action *odigosv1.Action, actionConfig ActionConfig, processorConfig any) (*odigosv1.Processor, error) {

	configJson, err := json.Marshal(processorConfig)
	if err != nil {
		return nil, err
	}

	collectorRoles := []odigosv1.CollectorsGroupRole{}
	for _, role := range actionConfig.CollectorRoles() {
		collectorRoles = append(collectorRoles, odigosv1.CollectorsGroupRole(role))
	}

	processor := odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      action.Name,
			Namespace: action.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: action.APIVersion,
					Kind:       action.Kind,
					Name:       action.Name,
					UID:        action.UID,
				},
			},
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            actionConfig.ProcessorType(),
			ProcessorName:   action.Spec.ActionName,
			Disabled:        action.Spec.Disabled,
			Notes:           action.Spec.Notes,
			Signals:         action.Spec.Signals,
			CollectorRoles:  collectorRoles,
			OrderHint:       actionConfig.OrderHint(),
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	return &processor, nil
}

func (r *ActionReconciler) reportReconciledToProcessorFailed(ctx context.Context, action *odigosv1.Action, reason odigosv1.ActionTransformedToProcessorReason, err error) error {
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               odigosv1.ActionTransformedToProcessorType,
		Status:             metav1.ConditionFalse,
		Reason:             string(reason),
		Message:            err.Error(),
		ObservedGeneration: action.Generation,
	})

	if changed {
		err := r.Status().Update(ctx, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ActionReconciler) reportProcessorNotRequired(ctx context.Context, action *odigosv1.Action) error {
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               odigosv1.ActionTransformedToProcessorType,
		Status:             metav1.ConditionTrue,
		Reason:             string(odigosv1.ActionTransformedToProcessorReasonProcessorNotRequired),
		Message:            "is not required for this action type.",
		ObservedGeneration: action.Generation,
	})

	if changed {
		err := r.Status().Update(ctx, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ActionReconciler) reportReconciledToProcessor(ctx context.Context, action *odigosv1.Action) error {
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               odigosv1.ActionTransformedToProcessorType,
		Status:             metav1.ConditionTrue,
		Reason:             string(odigosv1.ActionTransformedToProcessorReasonProcessorCreated),
		Message:            "The action has been reconciled to a processor resource.",
		ObservedGeneration: action.Generation,
	})

	if changed {
		logger := ctrl.LoggerFrom(ctx)
		logger.Info("Action reconciled successfully")
		err := r.Status().Update(ctx, action)
		if err != nil {
			logger.Error(err, "Failed to update action status to success")
			return err
		}
	}
	return nil
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	action := &odigosv1.Action{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If the action is a URL templatization action, we should not need to transform it to a processor CR.
	if action.Spec.URLTemplatization != nil {
		err = r.reportProcessorNotRequired(ctx, action)
		if err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, nil
	}

	processor, err := convertActionToProcessor(ctx, r.Client, action)
	if err != nil {
		logger.Error(err, "Failed to convert action to processor")
		err = r.reportReconciledToProcessorFailed(ctx, action, odigosv1.ActionTransformedToProcessorReasonFailedToTransformToProcessorReason, err)
		return utils.K8SUpdateErrorHandler(err) // return error of setting status, or nil if success (since the original error is not retryable and logged)
	}

	err = r.Patch(ctx, processor, client.Apply, client.FieldOwner(action.Name), client.ForceOwnership)
	if err != nil {
		statusErr := r.reportReconciledToProcessorFailed(ctx, action, odigosv1.ActionTransformedToProcessorReasonFailedToCreateProcessor, err)
		if statusErr == nil {
			return utils.K8SUpdateErrorHandler(err)
		} else {
			logger := ctrl.LoggerFrom(ctx)
			logger.Error(statusErr, "Failed to set status on action")
			return utils.K8SUpdateErrorHandler(err)
		}
	}

	err = r.reportReconciledToProcessor(ctx, action)
	return utils.K8SUpdateErrorHandler(err) // return error of setting status, or nil if success (since the original reconcile is successful)
}
