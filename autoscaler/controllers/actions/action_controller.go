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

const (
	urlTemplatizationProcessorName = "odigos-url-templatization"
	urlTemplatizationFieldOwner    = "url-templatization"

	// urlTemplatizationNamespaceSyncKey is a synthetic reconcile key used by the Processor watcher
	// to trigger a namespace-level sync when the Processor CR is deleted externally and no active
	// URLTemplatization actions exist to re-enqueue. This ensures the Processor is deleted if
	// no actions are present (orphan cleanup).
	urlTemplatizationNamespaceSyncKey = "odigos-url-templatization-ns-sync"

	// workloadConfigExtensionType is the OTel component type ID for the workload config extension.
	workloadConfigExtensionType = "odigos_workload_config"
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

// listActionsWithUrlTemplatization lists all non-disabled URLTemplatization Actions in the namespace.
func listActionsWithUrlTemplatization(ctx context.Context, k8sclient client.Client, namespace string) ([]odigosv1.Action, error) {
	var actionList odigosv1.ActionList
	if err := k8sclient.List(ctx, &actionList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	var result []odigosv1.Action
	for i := range actionList.Items {
		a := &actionList.Items[i]
		if a.Spec.URLTemplatization != nil && !a.Spec.Disabled {
			result = append(result, actionList.Items[i])
		}
	}
	return result, nil
}

// buildUrlTemplatizationProcessor builds the shared Processor CR for URL templatization.
// It has no OwnerReferences; its lifecycle is managed exclusively by syncUrlTemplatizationProcessorForNamespace.
func buildUrlTemplatizationProcessor(namespace string) (*odigosv1.Processor, error) {
	processorConfig := map[string]interface{}{
		"workload_config_extension": workloadConfigExtensionType,
	}
	configJSON, err := json.Marshal(processorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal url templatization processor config: %w", err)
	}

	return &odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      urlTemplatizationProcessorName,
			Namespace: namespace,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            "odigosurltemplate",
			ProcessorName:   "URL Templatization",
			Disabled:        false,
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleNodeCollector},
			OrderHint:       1,
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}, nil
}

// syncUrlTemplatizationProcessorForNamespace creates or deletes the shared URLTemplatization Processor CR
// based on whether any non-disabled URLTemplatization Actions exist in the namespace.
// This function is idempotent and safe to call on every reconcile.
func syncUrlTemplatizationProcessorForNamespace(ctx context.Context, r *ActionReconciler, namespace string) error {
	actions, err := listActionsWithUrlTemplatization(ctx, r.Client, namespace)
	if err != nil {
		return fmt.Errorf("failed to list url templatization actions: %w", err)
	}

	if len(actions) == 0 {
		// No active actions → delete the Processor CR if it exists.
		proc := &odigosv1.Processor{}
		proc.Namespace = namespace
		proc.Name = urlTemplatizationProcessorName
		return client.IgnoreNotFound(r.Client.Delete(ctx, proc))
	}

	// At least one active action → ensure the shared Processor CR exists.
	processor, err := buildUrlTemplatizationProcessor(namespace)
	if err != nil {
		return err
	}
	return r.Patch(ctx, processor, client.Apply, client.FieldOwner(urlTemplatizationFieldOwner), client.ForceOwnership)
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
		// Two cases reach here:
		// 1. A real Action was deleted — sync to remove the shared Processor CR if no others remain.
		// 2. Processor watcher found no live Actions (urlTemplatizationNamespaceSyncKey) —
		//    the Processor outlived all its Actions; sync deletes it.
		if syncErr := syncUrlTemplatizationProcessorForNamespace(ctx, r, req.Namespace); syncErr != nil {
			logger.Error(syncErr, "Failed to sync URL templatization processor for namespace after action delete")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// URL templatization: one shared Processor CR per namespace; rules come from extension in node collector.
	if action.Spec.URLTemplatization != nil {
		if err := syncUrlTemplatizationProcessorForNamespace(ctx, r, action.Namespace); err != nil {
			logger.Error(err, "Failed to sync URL templatization processor")
			return ctrl.Result{}, err
		}
		err = r.reportReconciledToProcessor(ctx, action)
		return utils.K8SUpdateErrorHandler(err)
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
