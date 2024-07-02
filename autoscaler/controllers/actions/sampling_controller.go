package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sampling "github.com/odigos-io/odigos/autoscaler/controllers/actions/sampling"
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

type OdigosSamplingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// Processor types
	SamplingProcessorType = "odigossampling"
	GroupByTraceType      = "groupbytrace"
)

func (r *OdigosSamplingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Sampling related actions")

	err := r.syncSamplingProcessors(ctx, req, logger)

	if err != nil {
		logger.V(0).Error(err, "Failed to reconcile sampling related actions")
	}

	return ctrl.Result{}, client.IgnoreNotFound(err)
}

func (r *OdigosSamplingReconciler) syncSamplingProcessors(ctx context.Context, req ctrl.Request, logger logr.Logger) error {
	desiredActions, err := r.getRelevantActions(ctx, req.Namespace)
	if err != nil {
		return err
	}

	if !r.isRelevantActions(desiredActions) {
		return nil
	}

	logger.V(0).Info("Sync sampling processor with desired actions")
	return r.syncOdigosSamplingProcessor(ctx, desiredActions, req.Namespace)
}

func (r *OdigosSamplingReconciler) getRelevantActions(ctx context.Context, namespace string) (map[reflect.Type][]metav1.Object, error) {
	relevantActions := make(map[reflect.Type][]metav1.Object)

	for actionType, handler := range sampling.SamplingSupportedActions {
		actions, err := handler.List(ctx, r.Client, namespace)
		if err != nil {
			return nil, err
		}

		filteredActions := r.filterActions(ctx, actions, handler)
		if len(filteredActions) > 0 {
			relevantActions[actionType] = filteredActions
		}
	}

	return relevantActions, nil
}

func (r *OdigosSamplingReconciler) syncOdigosSamplingProcessor(ctx context.Context, relevantActions map[reflect.Type][]metav1.Object, namespace string) error {
	var (
		actionsReferences    []metav1.OwnerReference
		globalActionsRules   []sampling.Rule
		endpointActionsRules []sampling.Rule
	)

	for actionType, actions := range relevantActions {

		handler := sampling.SamplingSupportedActions[actionType]

		for _, action := range actions {
			actionsReferences = append(actionsReferences, handler.GetActionReference(action))

			actionScope := handler.GetActionScope(action)
			if actionScope == "global" {
				globalActionsRules = append(globalActionsRules, handler.GetRuleConfig(action)...)
			}
			if actionScope == "endpoint" {
				endpointActionsRules = append(endpointActionsRules, handler.GetRuleConfig(action)...)
			}

			r.ReportReconciledToProcessor(ctx, action)
		}
	}

	samplingConf := sampling.SamplingConfig{
		EndpointRules: endpointActionsRules,
		GlobalRules:   globalActionsRules,
	}

	samplingConfigJson, err := json.Marshal(samplingConf)
	if err != nil {
		return err
	}

	samplingProcessor := &v1.Processor{
		TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "sampling-processor",
			Namespace:       namespace,
			OwnerReferences: actionsReferences,
		},
		Spec: v1.ProcessorSpec{
			Type:            SamplingProcessorType,
			ProcessorName:   SamplingProcessorType,
			Disabled:        false, // In case related actions are disabled, the processor won't be created
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  []v1.CollectorsGroupRole{v1.CollectorsGroupRoleClusterGateway},
			OrderHint:       -24,
			ProcessorConfig: runtime.RawExtension{Raw: samplingConfigJson},
		},
	}

	groupByTraceProcessor := r.getGroupByTraceProcessor(namespace, actionsReferences)
	if err := r.Patch(ctx, groupByTraceProcessor, client.Apply, client.FieldOwner("groupbytrace"), client.ForceOwnership); err != nil {
		return err
	}

	return r.Patch(ctx, samplingProcessor, client.Apply, client.FieldOwner("sampling-processor"), client.ForceOwnership)
}

func (r *OdigosSamplingReconciler) ReportReconciledToProcessorFailed(ctx context.Context, action metav1.Object, reason, msg string) error {
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

func (r *OdigosSamplingReconciler) ReportReconciledToProcessor(ctx context.Context, action metav1.Object) error {
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

func (r *OdigosSamplingReconciler) filterActions(ctx context.Context, actions []metav1.Object, handler sampling.ActionHandler) []metav1.Object {
	logger := log.FromContext(ctx)

	var filteredActions []metav1.Object
	for _, action := range actions {

		// filter disabled actions
		if handler.IsActionDisabled(action) {
			continue
		}
		// filter actions with invalid configuration
		if err := handler.ValidateRuleConfig(handler.GetRuleConfig(action)); err != nil {
			logger.V(0).Error(err, "Failed to validate rule config")
			r.ReportReconciledToProcessorFailed(ctx, action, "FailedToTransformToProcessorReason", err.Error())
			continue
		}

		filteredActions = append(filteredActions, action)
	}
	return filteredActions
}

func (r *OdigosSamplingReconciler) isRelevantActions(desiredActions map[reflect.Type][]metav1.Object) bool {
	for _, actions := range desiredActions {
		if len(actions) > 0 {
			return true
		}
	}
	return false
}

func getConditions(obj metav1.Object) (*[]metav1.Condition, error) {
	if obj == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{}, "no sampling related actions found")
	}

	val := reflect.ValueOf(obj).Elem().FieldByName("Status").FieldByName("Conditions")
	if !val.IsValid() {
		return nil, fmt.Errorf("conditions field not found in %T", obj)
	}

	conditions := val.Addr().Interface().(*[]metav1.Condition)

	return conditions, nil
}

func (r *OdigosSamplingReconciler) getGroupByTraceProcessor(namespace string, actionsReferences []metav1.OwnerReference) *v1.Processor {

	grouByAttributeConfig := sampling.GroupByTraceConfig{
		WaitDuration: sampling.DefaultWaitDuraiton,
	}
	groupByTraceConfigJson, err := json.Marshal(grouByAttributeConfig)
	if err != nil {
		return nil
	}

	return &v1.Processor{
		TypeMeta: metav1.TypeMeta{APIVersion: "odigos.io/v1alpha1", Kind: "Processor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "groupbytrace-processor",
			Namespace:       namespace,
			OwnerReferences: actionsReferences,
		},
		Spec: v1.ProcessorSpec{
			Type:            GroupByTraceType,
			ProcessorName:   GroupByTraceType,
			Disabled:        false, // In case related actions are disabled, the processor wont be created
			Signals:         []common.ObservabilitySignal{common.TracesObservabilitySignal},
			CollectorRoles:  []v1.CollectorsGroupRole{v1.CollectorsGroupRoleClusterGateway},
			OrderHint:       -25,
			ProcessorConfig: runtime.RawExtension{Raw: groupByTraceConfigJson},
		},
	}
}
