package actions

import (
	"context"
	"encoding/json"
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	actionscatalog "github.com/odigos-io/odigos/actions"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	actionutil "github.com/odigos-io/odigos/k8sutils/pkg/action"
	odgiosK8s "github.com/odigos-io/odigos/k8sutils/pkg/conditions"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/status"
	actionstatus "github.com/odigos-io/odigos/status/action/generated"
	processorstatus "github.com/odigos-io/odigos/status/processor/generated"
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

	if action.Spec.RenameAttribute != nil {
		config, err := renameAttributeConfig(action.Spec.RenameAttribute.Renames, action.Spec.Signals)
		if err != nil {
			return nil, err
		}
		return convertToDefaultProcessor(action, action.Spec.RenameAttribute, config)
	}

	if action.Spec.ExtractAttribute != nil {
		config, err := extractAttributeConfig(action.Spec.ExtractAttribute)
		if err != nil {
			return nil, err
		}
		return convertToDefaultProcessor(action, action.Spec.ExtractAttribute, config)
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

	return nil, errors.New("no supported action found in resource")
}

// supportedSignals filters the signals requested by the action down to the ones the action catalog
// declares its processor can consume. The Action CRD accepts any signal, but a processor placed in a
// pipeline its factory does not implement makes the collector fail to build that pipeline
// ("telemetry type is not supported") and crash-loop, so unsupported signals must never reach the
// Processor CR. Actions with no catalog entry are left as-is, there is nothing authoritative to
// filter against.
func supportedSignals(action *odigosv1.Action, signals []common.ObservabilitySignal) []common.ObservabilitySignal {
	manifest, found := actionscatalog.GetActionByType(actionutil.CatalogType(action))
	if !found {
		return signals
	}

	supported := map[common.ObservabilitySignal]bool{
		common.TracesObservabilitySignal:   manifest.Spec.Signals.Traces.Supported,
		common.MetricsObservabilitySignal:  manifest.Spec.Signals.Metrics.Supported,
		common.LogsObservabilitySignal:     manifest.Spec.Signals.Logs.Supported,
		common.ProfilesObservabilitySignal: manifest.Spec.Signals.Profiles.Supported,
	}

	filtered := make([]common.ObservabilitySignal, 0, len(signals))
	for _, signal := range signals {
		if supported[signal] {
			filtered = append(filtered, signal)
		}
	}
	return filtered
}

// returns a processor object with:
// - ns and name similar to the action name
// - signals based on the action signals, limited to those its processor supports
// - owner reference to the action
// - type and order hint based on the function input
// - config based on the function input, stringified in JSON
// - collector roles from actionConfig.CollectorRoles()
func convertToDefaultProcessor(action *odigosv1.Action, actionConfig ActionConfig, processorConfig any) (*odigosv1.Processor, error) {

	configJson, err := json.Marshal(processorConfig)
	if err != nil {
		return nil, err
	}

	collectorRoles := make([]odigosv1.CollectorsGroupRole, 0, len(actionConfig.CollectorRoles()))
	for _, r := range actionConfig.CollectorRoles() {
		collectorRoles = append(collectorRoles, odigosv1.CollectorsGroupRole(r))
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
			Signals:         supportedSignals(action, action.Spec.Signals),
			CollectorRoles:  collectorRoles,
			OrderHint:       actionConfig.OrderHint(),
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	return &processor, nil
}

func (r *ActionReconciler) reportReconciledToProcessorFailed(ctx context.Context, action *odigosv1.Action, reason status.Reason, reconcileErr error) error {
	message, _ := status.RenderMessage(reason, actionstatus.TransformedToProcessorMessageParams{
		Error: reconcileErr.Error(),
	})
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               actionstatus.TransformedToProcessorType,
		Status:             reason.K8sConditionStatus,
		Reason:             reason.Name,
		Message:            message,
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
	reason := actionstatus.TransformedToProcessorProcessorCreated
	if action.Spec.Disabled {
		reason = actionstatus.TransformedToProcessorActionDisabled
	}
	message, _ := status.RenderMessage(reason, nil)
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               actionstatus.TransformedToProcessorType,
		Status:             reason.K8sConditionStatus,
		Reason:             reason.Name,
		Message:            message,
		ObservedGeneration: action.Generation,
	})

	if changed {
		logger := commonlogger.FromContext(ctx)
		logger.Info("Action reconciled successfully")
		err := r.Status().Update(ctx, action)
		if err != nil {
			logger.Error(err, "Failed to update action status after reconcile")
			return err
		}
	}
	return nil
}

// clearTransformedToProcessor removes TransformedToProcessor when this action is not
// reconciled to a Processor CR (e.g. odigosConfigExtension actions).
func (r *ActionReconciler) clearTransformedToProcessor(ctx context.Context, action *odigosv1.Action) error {
	if meta.FindStatusCondition(action.Status.Conditions, actionstatus.TransformedToProcessorType) == nil {
		return nil
	}
	if !meta.RemoveStatusCondition(&action.Status.Conditions, actionstatus.TransformedToProcessorType) {
		return nil
	}
	return r.Status().Update(ctx, action)
}

// syncAddedToCollectorConfigFromOwnedProcessor mirrors AddedToCollectorConfig from the
// 1:1 owned Processor (same namespace/name as the Action). Shared processors are out of scope.
func (r *ActionReconciler) syncAddedToCollectorConfigFromOwnedProcessor(ctx context.Context, action *odigosv1.Action) error {
	processor := &odigosv1.Processor{}
	err := r.Get(ctx, client.ObjectKey{Namespace: action.Namespace, Name: action.Name}, processor)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.clearAddedToCollectorConfig(ctx, action)
		}
		return err
	}

	cond := meta.FindStatusCondition(processor.Status.Conditions, processorstatus.AddedToCollectorConfigType)
	if cond == nil {
		return r.clearAddedToCollectorConfig(ctx, action)
	}

	reason, ok := actionstatus.AddedToCollectorConfigReasonByName(cond.Reason)
	if !ok {
		return r.clearAddedToCollectorConfig(ctx, action)
	}

	message := cond.Message
	if message == "" {
		message, _ = status.RenderMessage(reason, nil)
	}
	return odgiosK8s.UpdateStatusConditions(ctx, r.Client, action, &action.Status.Conditions,
		reason.K8sConditionStatus,
		actionstatus.AddedToCollectorConfigType,
		reason.Name,
		message,
	)
}

func (r *ActionReconciler) clearAddedToCollectorConfig(ctx context.Context, action *odigosv1.Action) error {
	if meta.FindStatusCondition(action.Status.Conditions, actionstatus.AddedToCollectorConfigType) == nil {
		return nil
	}
	if !meta.RemoveStatusCondition(&action.Status.Conditions, actionstatus.AddedToCollectorConfigType) {
		return nil
	}
	return r.Status().Update(ctx, action)
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)

	action := &odigosv1.Action{}
	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		if apierrors.IsNotFound(err) {

			// Reconcile after delete: sync shared processors so they are removed when no matching actions remain.
			if syncErr := SyncUrlTemplatizationProcessor(ctx, r.Client, URLTemplatizationSyncApplyFull); syncErr != nil {
				logger.Error(syncErr, "sync URL templatization processor after action delete failed")
				return ctrl.Result{}, syncErr
			}

			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// ToDo: once we add a mutating webhook to actions, we will inject labels for shared-processor
	// actions and then filter which syncs to run. Right now running for every Action so shared
	// processor lifecycle is correctly managed.
	if err := SyncUrlTemplatizationProcessor(ctx, r.Client, URLTemplatizationSyncCreateIfMissing); err != nil {
		logger.Error(err, "sync URL templatization processor failed")
		return ctrl.Result{}, err
	}
	if action.Spec.URLTemplatization != nil {
		err = r.reportReconciledToProcessor(ctx, action)
		return utils.K8SUpdateErrorHandler(err)
	}

	// Config-extension actions (including PiiMasking) are applied via collector config, not Processor CRs.
	if actionutil.IsConfigExtension(action) {
		return utils.K8SUpdateErrorHandler(r.clearTransformedToProcessor(ctx, action))
	}

	processor, err := convertActionToProcessor(ctx, r.Client, action)
	if err != nil {
		logger.Error(err, "Failed to convert action to processor")
		err = r.reportReconciledToProcessorFailed(ctx, action, actionstatus.TransformedToProcessorFailedToTransformToProcessor, err)
		return utils.K8SUpdateErrorHandler(err) // return error of setting status, or nil if success (since the original error is not retryable and logged)
	}

	err = r.Patch(ctx, processor, client.Apply, client.FieldOwner(action.Name), client.ForceOwnership)
	if err != nil {
		statusErr := r.reportReconciledToProcessorFailed(ctx, action, actionstatus.TransformedToProcessorFailedToCreateProcessor, err)
		if statusErr == nil {
			return utils.K8SUpdateErrorHandler(err)
		} else {
			logger := commonlogger.FromContext(ctx)
			logger.Error(statusErr, "Failed to set status on action")
			return utils.K8SUpdateErrorHandler(err)
		}
	}

	err = r.reportReconciledToProcessor(ctx, action)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}

	// Mirror AddedToCollectorConfig from the 1:1 owned Processor when present.
	// Usually empty on first reconcile; Owns(Processor) re-triggers after collector sync.
	if processor.Name != action.Name {
		return ctrl.Result{}, nil
	}
	return utils.K8SUpdateErrorHandler(r.syncAddedToCollectorConfigFromOwnedProcessor(ctx, action))
}
