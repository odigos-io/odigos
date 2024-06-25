package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sampling "github.com/odigos-io/odigos/autoscaler/controllers/actions/tailsampling"
	commonproc "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TailSamplingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// Processor types
	tailSamplingProcessorType  = "tail_sampling"
	probabilisticProcessorType = "probabilistic_sampler"
)

func (r *TailSamplingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Sampling related actions")

	err := r.syncSamplingProcessors(ctx, req, logger)

	if err != nil {
		logger.V(0).Error(err, "Failed to reconcile sampling related actions")
	}

	return ctrl.Result{}, client.IgnoreNotFound(err)
}

func (r *TailSamplingReconciler) syncSamplingProcessors(ctx context.Context, req ctrl.Request, logger logr.Logger) error {
	desiredActions, err := r.getRelevantActions(ctx, req.Namespace)
	if err != nil {
		return err
	}

	if !r.isRelevantActions(desiredActions) {
		return r.deleteSamplingProcessorsIfExists(ctx, req.Namespace)
	}

	if r.isOnlySamplingActionProbablistic(desiredActions) {
		logger.V(0).Info("Sync dedicated probabilistic sampler processor")
		return r.syncProbabilisticSamplingProcessor(ctx, desiredActions, req.Namespace)
	}

	logger.V(0).Info("Sync tail sampling processor with desired actions")
	return r.syncTailSamplingProcessor(ctx, desiredActions, req.Namespace)
}

func (r *TailSamplingReconciler) getRelevantActions(ctx context.Context, namespace string) (map[reflect.Type][]metav1.Object, error) {
	relevantActions := make(map[reflect.Type][]metav1.Object)

	for actionType, handler := range sampling.TailSamplingSupportedActions {
		if actions, err := handler.List(ctx, r.Client, namespace); err != nil {
			return nil, err
		} else {

			filteredActions := r.filterActions(ctx, actions, handler)
			if len(filteredActions) > 0 {
				relevantActions[actionType] = r.filterActions(ctx, actions, handler)
			}
		}
	}
	return relevantActions, nil
}

func (r *TailSamplingReconciler) syncProbabilisticSamplingProcessor(ctx context.Context, relevantActions map[reflect.Type][]metav1.Object, namespace string) error {
	probType := reflect.TypeOf(&actionv1.ProbabilisticSampler{})

	probabilisticActions := relevantActions[probType]

	handler := sampling.TailSamplingSupportedActions[probType]

	action, err := handler.SelectSampler(probabilisticActions)
	if err != nil {
		return err
	}

	cfg := handler.GetPolicyConfig(action)

	configJson, err := json.Marshal(cfg.Details)
	if err != nil {
		return err
	}

	r.reportActionStatus(ctx, probabilisticActions, action)

	probAction := action.(*actionv1.ProbabilisticSampler)

	processor := v1.Processor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Processor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "probabilistic-sampler-processor",
			Namespace: probAction.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				handler.GetActionReference(action),
			},
		},
		Spec: v1.ProcessorSpec{
			Type:            probabilisticProcessorType,
			ProcessorName:   probAction.Spec.ActionName,
			Disabled:        probAction.Spec.Disabled,
			Notes:           probAction.Spec.Notes,
			Signals:         probAction.Spec.Signals,
			CollectorRoles:  []v1.CollectorsGroupRole{v1.CollectorsGroupRoleNodeCollector},
			OrderHint:       1,
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	if err := r.Patch(ctx, &processor, client.Apply, client.FieldOwner(probAction.Name), client.ForceOwnership); err != nil {
		r.ReportReconciledToProcessorFailed(ctx, action, "FailedToCreateProcessorReason", err.Error())
		return err
	}

	// Prevents cases where all other tail sampling actions removed except from probabilistic sampler
	if err := commonproc.DeleteProcessorByType(ctx, r.Client, tailSamplingProcessorType, namespace); err != nil {
		return err
	}

	r.ReportReconciledToProcessor(ctx, action)
	return nil
}

func (r *TailSamplingReconciler) syncTailSamplingProcessor(ctx context.Context, relevantActions map[reflect.Type][]metav1.Object, namespace string) error {
	var (
		actionsReferences []metav1.OwnerReference
		actionsPolicies   []sampling.Policy
	)

	for actionType, actions := range relevantActions {

		handler := sampling.TailSamplingSupportedActions[actionType]

		if len(actions) > 0 {
			action, err := handler.SelectSampler(actions)
			if err != nil {
				return err
			}

			actionReference := handler.GetActionReference(action)
			actionsReferences = append(actionsReferences, actionReference)

			policyConfig := handler.GetPolicyConfig(action)
			actionsPolicies = append(actionsPolicies, policyConfig)

			r.reportActionStatus(ctx, actions, action)

		}
	}

	conf := sampling.TailSamplingConfig{
		Policies: actionsPolicies,
	}
	configJson, err := json.Marshal(conf)
	if err != nil {
		return err
	}

	processor := &v1.Processor{
		TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "tail-sampling-processor",
			Namespace:       namespace,
			OwnerReferences: actionsReferences,
		},
		Spec: v1.ProcessorSpec{
			Type:            tailSamplingProcessorType,
			ProcessorName:   tailSamplingProcessorType,
			Disabled:        false, // In case related actions are disabled, the processor wont be created
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  []v1.CollectorsGroupRole{v1.CollectorsGroupRoleClusterGateway},
			OrderHint:       -25,
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	if err := r.Patch(ctx, processor, client.Apply, client.FieldOwner("tail-sampling-controller"), client.ForceOwnership); err != nil {
		return err
	}

	// Delete ProbabilisticSampler processor to avoid cases where probabilistic sampler processor already exists
	return commonproc.DeleteProcessorByType(ctx, r.Client, probabilisticProcessorType, namespace)
}

func (r *TailSamplingReconciler) ReportReconciledToProcessorFailed(ctx context.Context, action metav1.Object, reason, msg string) error {
	conditions, err := getConditions(action)
	if err != nil {
		return err
	}

	changed := meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               "ActionTransformedToProcessorType",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            msg,
		ObservedGeneration: action.GetGeneration(),
	})

	if changed {
		return r.Status().Update(ctx, action.(client.Object))
	}
	return nil
}

func (r *TailSamplingReconciler) ReportReconciledToProcessor(ctx context.Context, action metav1.Object) error {
	conditions, err := getConditions(action)
	if err != nil {
		return err
	}

	changed := meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               "ActionTransformedToProcessorType",
		Status:             metav1.ConditionTrue,
		Reason:             "ProcessorCreatedReason",
		Message:            "The action has been reconciled to a processor resource.",
		ObservedGeneration: action.GetGeneration(),
	})

	if changed {
		if err := r.Status().Update(ctx, action.(client.Object)); err != nil {
			return err
		}
	}
	return nil
}

// In case tail sampling related actions exists, append it to tail_sampling processor
// otherwise, create a dedicated probablistic sampler processor
func (r *TailSamplingReconciler) isOnlySamplingActionProbablistic(relevantActions map[reflect.Type][]metav1.Object) bool {
	_, hasProbabilistic := relevantActions[reflect.TypeOf(&actionv1.ProbabilisticSampler{})]
	return hasProbabilistic && len(relevantActions) == 1
}

// Report the status of the actions, if there are multiple actions:
// Report Failed for all actions except the selected one
func (r *TailSamplingReconciler) reportActionStatus(ctx context.Context, filteredActions []metav1.Object, action metav1.Object) {
	for _, act := range filteredActions {
		if act.GetUID() != action.GetUID() {
			r.ReportReconciledToProcessorFailed(ctx, act, "FailedToTransformToProcessorReason", "Multiple similar actions found, selected the most appropriate one.")
		} else {
			r.ReportReconciledToProcessor(ctx, action)
		}
	}
}

func (r *TailSamplingReconciler) filterActions(ctx context.Context, actions []metav1.Object, handler sampling.ActionHandler) []metav1.Object {
	logger := log.FromContext(ctx)

	var filteredActions []metav1.Object
	for _, action := range actions {

		// filter disabled actions
		if handler.IsActionDisabled(action) {
			continue
		}
		// filter actions with invalid configuration
		if err := handler.ValidatePolicyConfig(handler.GetPolicyConfig(action)); err != nil {
			logger.V(0).Error(err, "Failed to validate policy config")
			r.ReportReconciledToProcessorFailed(context.Background(), action, "FailedToTransformToProcessorReason", err.Error())
			continue
		}

		filteredActions = append(filteredActions, action)
	}
	return filteredActions
}

func (r *TailSamplingReconciler) isRelevantActions(desiredActions map[reflect.Type][]metav1.Object) bool {
	for _, actions := range desiredActions {
		if len(actions) > 0 {
			return true
		}
	}
	return false
}

func (r *TailSamplingReconciler) deleteSamplingProcessorsIfExists(ctx context.Context, namespace string) error {
	if err := commonproc.DeleteProcessorByType(ctx, r.Client, tailSamplingProcessorType, namespace); err != nil {
		return err
	}
	return commonproc.DeleteProcessorByType(ctx, r.Client, probabilisticProcessorType, namespace)
}

func getConditions(obj metav1.Object) (*[]metav1.Condition, error) {
	if obj == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{}, "no tail sampling related actions found")
	}

	val := reflect.ValueOf(obj).Elem().FieldByName("Status").FieldByName("Conditions")
	if !val.IsValid() {
		return nil, fmt.Errorf("conditions field not found in %T", obj)
	}

	conditions := val.Addr().Interface().(*[]metav1.Condition)

	return conditions, nil
}
