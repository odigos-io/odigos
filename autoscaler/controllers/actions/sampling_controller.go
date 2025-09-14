package actions

import (
	"context"
	"encoding/json"
	"reflect"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	sampling "github.com/odigos-io/odigos/autoscaler/controllers/actions/sampling"
	"github.com/odigos-io/odigos/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DEPRECATED: Use odigosv1.Action instead
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
	logger.V(0).Info("Reconciling Sampling action")
	logger.V(0).Info("WARNING: Sampling action is deprecated and will be removed in a future version. Migrate to odigosv1.Action instead.")

	// Find the action type and handler for this request
	var action metav1.Object
	var handler sampling.ActionHandler
	var err error

	// Try to find the action in each supported type
	for actionType, actionHandler := range sampling.SamplingSupportedActions {
		// Create a new instance of the action type
		actionObj := reflect.New(actionType.Elem()).Interface().(metav1.Object)

		// Try to get the specific action
		err = r.Get(ctx, req.NamespacedName, actionObj.(client.Object))
		if err == nil {
			action = actionObj
			handler = actionHandler
			break
		}
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}

	if action == nil {
		// Action not found in any supported type
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Migrate to odigosv1.Action
	migratedActionName := v1.ActionMigratedLegacyPrefix + action.GetName()
	odigosAction := &v1.Action{}
	err = r.Get(ctx, client.ObjectKey{Name: migratedActionName, Namespace: action.GetNamespace()}, odigosAction)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		// Action doesn't exist, create new one
		odigosAction = r.createMigratedAction(action, handler, migratedActionName)
		err = r.Create(ctx, odigosAction)
		if err != nil {
			return ctrl.Result{}, err
		}
		action.SetOwnerReferences(append(action.GetOwnerReferences(), metav1.OwnerReference{
			APIVersion: "odigos.io/v1alpha1",
			Kind:       "Action",
			Name:       odigosAction.Name,
			UID:        odigosAction.UID,
		}))
		err = r.Update(ctx, action.(client.Object))
		return ctrl.Result{}, err
	}
	logger.V(0).Info("Migrated Action already exists, skipping update")
	return ctrl.Result{}, nil
}

func (r *OdigosSamplingReconciler) createMigratedAction(action metav1.Object, handler sampling.ActionHandler, migratedActionName string) *v1.Action {
	// Use the existing ConvertLegacyToAction method from the handler
	convertedAction := handler.ConvertLegacyToAction(action)

	// Cast to odigosv1.Action
	odigosAction := convertedAction.(*v1.Action)
	odigosAction.ObjectMeta.Name = migratedActionName

	return odigosAction
}

// samplersConfig converts the SamplersConfig to the appropriate processor configuration
func samplersConfig(ctx context.Context, client client.Client, namespace string) (any, []metav1.OwnerReference, error) {
	desiredActions, err := getRelevantActions(ctx, client, namespace)
	if err != nil {
		return nil, nil, err
	}
	if len(desiredActions) == 0 {
		return nil, nil, nil
	}

	var (
		actionsReferences []metav1.OwnerReference
		globalRules       []sampling.Rule
		serviceRules      []sampling.Rule
		endpointRules     []sampling.Rule
	)

	// Track owner references by UID to avoid duplicates
	ownerRefsByUID := make(map[string]metav1.OwnerReference)

	for actionType, actions := range desiredActions {
		handler := sampling.SamplingSupportedActions[actionType]

		for _, action := range actions {
			ownerRef := handler.GetActionReference(action)
			// Only add if we haven't seen this UID before
			if ownerRef.UID != "" {
				ownerRefsByUID[string(ownerRef.UID)] = ownerRef
			}

			actionScope := handler.GetActionScope(action)
			switch actionScope {
			case "global":
				globalRules = append(globalRules, handler.GetRuleConfig(action)...)
			case "service":
				serviceRules = append(serviceRules, handler.GetRuleConfig(action)...)
			case "endpoint":
				endpointRules = append(endpointRules, handler.GetRuleConfig(action)...)
			}

		}
	}

	// Convert map to slice
	for _, ownerRef := range ownerRefsByUID {
		actionsReferences = append(actionsReferences, ownerRef)
	}

	samplingConf := sampling.SamplingConfig{
		GlobalRules:   globalRules,
		ServiceRules:  serviceRules,
		EndpointRules: endpointRules,
	}

	return samplingConf, actionsReferences, nil
}

func getRelevantActions(ctx context.Context, client client.Client, namespace string) (map[reflect.Type][]metav1.Object, error) {
	logger := log.FromContext(ctx)
	relevantActions := make(map[reflect.Type][]metav1.Object)

	for actionType, handler := range sampling.SamplingSupportedActions {
		actions, err := handler.List(ctx, client, namespace)
		if err != nil {
			return nil, err
		}

		var filteredActions []metav1.Object
		for _, action := range actions {

			// filter disabled actions
			if handler.IsActionDisabled(action) {
				continue
			}
			// filter actions with invalid configuration
			if err := handler.ValidateRuleConfig(handler.GetRuleConfig(action)); err != nil {
				logger.V(0).Error(err, "Failed to validate rule config")
				//ReportReconciledToProcessorFailed(ctx, client, action, "FailedToTransformToProcessorReason", err.Error())
				continue
			}

			filteredActions = append(filteredActions, action)
		}

		if len(filteredActions) > 0 {
			relevantActions[actionType] = filteredActions
		}
	}

	return relevantActions, nil
}

func getGroupByTraceProcessor(namespace string, actionsReferences []metav1.OwnerReference) *v1.Processor {

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
