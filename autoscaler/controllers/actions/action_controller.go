package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

const (
	// urlTemplatizationNamespaceSyncKey is a synthetic reconcile key used by the Processor watcher
	// to trigger a namespace-level sync when the Processor CR is deleted externally and no active
	// URLTemplatization actions exist to re-enqueue. This ensures the Processor is deleted if
	// no actions are present (orphan cleanup).
	// Keep this value invalid as a k8s object name (underscores) to avoid collisions with real Actions.
	urlTemplatizationNamespaceSyncKey = "__odigos_url_templatization_ns_sync__"

	// Label set on Actions that have URLTemplatization and are not disabled. Used to list only
	// relevant Actions in syncUrlTemplatizationProcessorForNamespace (server-side filter).
	urlTemplatizationLabelKey   = "odigos.io/url-templatization"
	urlTemplatizationLabelValue = "true"
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

// listActionsWithUrlTemplatization lists Actions in the namespace that have the URL-templatization
// label (odigos.io/url-templatization=true). We set that label only when Spec.URLTemplatization != nil
// and !Spec.Disabled, so this is a server-side filter for relevant Actions and avoids listing all Actions.
func listActionsWithUrlTemplatization(ctx context.Context, k8sclient client.Client, namespace string) ([]odigosv1.Action, error) {
	var actionList odigosv1.ActionList
	if err := k8sclient.List(ctx, &actionList, client.InNamespace(namespace), client.MatchingLabels{
		urlTemplatizationLabelKey: urlTemplatizationLabelValue,
	}); err != nil {
		return nil, err
	}
	return actionList.Items, nil
}

// buildUrlTemplatizationProcessor builds the shared Processor CR for URL templatization.
// It has no OwnerReferences; its lifecycle is managed exclusively by syncUrlTemplatizationProcessorForNamespace.
func buildUrlTemplatizationProcessor(namespace string) (*odigosv1.Processor, error) {
	processorConfig := map[string]interface{}{
		"workload_config_extension": commonconsts.OdigosConfigK8sExtensionType,
	}
	configJSON, err := json.Marshal(processorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal url templatization processor config: %w", err)
	}

	urlTemplatizationConfig := actions.URLTemplatizationConfig{}
	collectorRoles := make([]odigosv1.CollectorsGroupRole, 0, len(urlTemplatizationConfig.CollectorRoles()))
	for _, role := range urlTemplatizationConfig.CollectorRoles() {
		collectorRoles = append(collectorRoles, odigosv1.CollectorsGroupRole(role))
	}

	return &odigosv1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      commonconsts.URLTemplatizationProcessorName,
			Namespace: namespace,
		},
		Spec: odigosv1.ProcessorSpec{
			Type:            urlTemplatizationConfig.ProcessorType(),
			ProcessorName:   "URL Templatization",
			Disabled:        false,
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  collectorRoles,
			OrderHint:       urlTemplatizationConfig.OrderHint(),
			ProcessorConfig: runtime.RawExtension{Raw: configJSON},
		},
	}, nil
}

// ensureUrlTemplatizationLabel sets or removes the odigos.io/url-templatization label on the Action
// so that listActionsWithUrlTemplatization can use a label selector and only fetch relevant Actions.
// Should be called when reconciling an Action that has or had URLTemplatization.
func ensureUrlTemplatizationLabel(ctx context.Context, r *ActionReconciler, action *odigosv1.Action) error {
	wantLabel := action.Spec.URLTemplatization != nil && !action.Spec.Disabled
	hasLabel := action.Labels != nil && action.Labels[urlTemplatizationLabelKey] == urlTemplatizationLabelValue
	if wantLabel == hasLabel {
		return nil
	}
	actionCopy := action.DeepCopy()
	if actionCopy.Labels == nil {
		actionCopy.Labels = make(map[string]string)
	}
	if wantLabel {
		actionCopy.Labels[urlTemplatizationLabelKey] = urlTemplatizationLabelValue
	} else {
		delete(actionCopy.Labels, urlTemplatizationLabelKey)
	}
	return r.Patch(ctx, actionCopy, client.MergeFrom(action))
}

// ensureUrlTemplatizationProcessorExists creates the shared URLTemplatization Processor CR in the namespace if it does not exist.
// Use when the caller already knows at least one non-disabled URLTemplatization Action exists (avoids a List).
// Idempotent: if the Processor already exists (e.g. from a concurrent reconcile), returns nil.
func ensureUrlTemplatizationProcessorExists(ctx context.Context, r *ActionReconciler, namespace string) error {
	existing := &odigosv1.Processor{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: commonconsts.URLTemplatizationProcessorName}, existing)
	if err == nil {
		return nil // already exists
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	processor, err := buildUrlTemplatizationProcessor(namespace)
	if err != nil {
		return err
	}
	if createErr := r.Create(ctx, processor); createErr != nil {
		return createErr
	}
	ctrl.LoggerFrom(ctx).Info("Created URL templatization Processor", "namespace", namespace)
	return nil
}

// syncUrlTemplatizationProcessorForNamespace creates or deletes the shared URLTemplatization Processor CR
// based on whether any relevant Actions (with URL-templatization label) exist in the namespace.
// Gets the Processor first; if it already exists and there is at least one Action, does nothing (no Patch).
// This function is idempotent and safe to call on every reconcile.
func syncUrlTemplatizationProcessorForNamespace(ctx context.Context, r *ActionReconciler, namespace string) error {
	actions, err := listActionsWithUrlTemplatization(ctx, r.Client, namespace)
	if err != nil {
		return fmt.Errorf("failed to list url templatization actions: %w", err)
	}

	if len(actions) == 0 {
		// No relevant actions → delete the Processor CR if it exists.
		proc := &odigosv1.Processor{}
		proc.Namespace = namespace
		proc.Name = commonconsts.URLTemplatizationProcessorName
		err := client.IgnoreNotFound(r.Client.Delete(ctx, proc))
		if err == nil {
			ctrl.LoggerFrom(ctx).Info("Deleted URL templatization Processor (no actions in namespace)", "namespace", namespace)
		}
		return err
	}

	// At least one relevant action. Avoid Patch if the Processor already exists.
	existing := &odigosv1.Processor{}
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: commonconsts.URLTemplatizationProcessorName}, existing)
	if err == nil {
		// Processor exists; nothing to do.
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get processor: %w", err)
	}
	// Processor not found → create it.
	ctrl.LoggerFrom(ctx).Info("URL templatization Processor not found, creating", "namespace", namespace)
	return ensureUrlTemplatizationProcessorExists(ctx, r, namespace)
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

	// Synthetic namespace-level reconcile request (enqueued from the Processor watcher).
	// This path is independent from Action object lookup and is used for orphan cleanup/retry.
	if req.Name == urlTemplatizationNamespaceSyncKey {
		if err := syncUrlTemplatizationProcessorForNamespace(ctx, r, req.Namespace); err != nil {
			logger.Error(err, "Failed to sync URL templatization processor for synthetic namespace reconcile")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

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

	// Keep URL-templatization label in sync so listActionsWithUrlTemplatization can use a label selector.
	if err := ensureUrlTemplatizationLabel(ctx, r, action); err != nil {
		logger.Error(err, "Failed to update URL templatization label on action")
		return ctrl.Result{}, err
	}

	// URL templatization: one shared Processor CR per namespace; per-workload rules are served
	// at runtime by the odigos_config_k8s extension running in the cluster gateway.
	if action.Spec.URLTemplatization != nil {
		if action.Spec.Disabled {
			// Must list to know if any other non-disabled Actions remain; may need to delete Processor.
			if err := syncUrlTemplatizationProcessorForNamespace(ctx, r, action.Namespace); err != nil {
				logger.Error(err, "Failed to sync URL templatization processor")
				return ctrl.Result{}, err
			}
		} else {
			// At least one non-disabled Action exists (this one); ensure Processor exists without listing.
			if err := ensureUrlTemplatizationProcessorExists(ctx, r, action.Namespace); err != nil {
				logger.Error(err, "Failed to ensure URL templatization processor")
				return ctrl.Result{}, err
			}
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
