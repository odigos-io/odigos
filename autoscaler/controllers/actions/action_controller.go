package actions

import (
	"context"
	"encoding/json"
	"errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActionReconciler struct {
	client.Client
}

// TODO: should we import the processor and use it's type and validation?
type odigosurltemplateProcessorConfig struct {
	// The processor by default will templatize numbers and uuids.
	// This will cover some cases, but if id is a name, pattern with letters, internal representation, etc
	// those cannot be detected deterministically and might create high cardinality in span names and low cardinality attributes.
	// The TemplatizationRules is a list of path templatizations specific rules that will be applied and taken if matched.
	// A rule is a pattern for a path that is composed of multiple path segments separated by "/".
	// each segment can be a string or a template variable.
	// strings are matched as is and are used in the template to replace the segment.
	// templatization segments like this: "/{name:regex}" and are used to match and replace the segment with the name.
	// e.g. "/v1/{foo:regex}/bar/{baz}" will match "/v1/123/bar/456" and will replace it with "/v1/:foo/bar/:baz"
	// if regex is not used, the segment will always match and replaced with the name.
	// if regex is used, and does not match, the segment will be skipped and will not take effect.
	TemplatizationRules []string `json:"templatization_rules"`
}

// giving an action, return it's specific processor details
func actionProcessorDetails(action *odigosv1.Action) (string, int, any, error) {
	if action.Spec.CalculateUrlTemplate != nil {
		odigosurltemplateProcessorConfig := odigosurltemplateProcessorConfig{
			TemplatizationRules: action.Spec.CalculateUrlTemplate.TemplatizationRules,
		}
		return "odigosurltemplate", 3, odigosurltemplateProcessorConfig, nil
	}

	return "", 0, nil, errors.New("no supported action found in resource")
}

// returns a processor object with:
// - ns and name similar to the action name
// - signals based on the action signals
// - owner reference to the action
// - type and order hint based on the function input
// - config based on the function input, stringified in JSON
// - collector roles set to ClusterGateway
func convertToProcessor(action *odigosv1.Action, processorType string, orderHint int, processorConfig any) (*odigosv1.Processor, error) {

	configJson, err := json.Marshal(processorConfig)
	if err != nil {
		return nil, err
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
			Type:            processorType,
			ProcessorName:   action.Spec.ActionName,
			Disabled:        action.Spec.Disabled,
			Notes:           action.Spec.Notes,
			Signals:         action.Spec.Signals,
			CollectorRoles:  []odigosv1.CollectorsGroupRole{odigosv1.CollectorsGroupRoleClusterGateway},
			OrderHint:       orderHint,
			ProcessorConfig: runtime.RawExtension{Raw: configJson},
		},
	}

	return &processor, nil
}

func (r *ActionReconciler) reportReconciledToProcessorFailed(ctx context.Context, action *odigosv1.Action, reason string, err error) error {
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               ActionTransformedToProcessorType,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
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

func (r *ActionReconciler) reportReconciledToProcessor(ctx context.Context, action *odigosv1.Action) error {
	changed := meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:               ActionTransformedToProcessorType,
		Status:             metav1.ConditionTrue,
		Reason:             ProcessorCreatedReason,
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

	processorType, orderHint, config, err := actionProcessorDetails(action)
	if err != nil {
		logger.Error(err, "Failed to get processor details from action")
		err = r.reportReconciledToProcessorFailed(ctx, action, FailedToTransformToProcessorReason, err)
		return utils.K8SUpdateErrorHandler(err) // return error of setting status, or nil if success (since the original error is not retryable and logged)
	}

	processor, err := convertToProcessor(action, processorType, orderHint, config)
	if err != nil {
		logger.Error(err, "Failed to convert action to processor")
		err = r.reportReconciledToProcessorFailed(ctx, action, FailedToTransformToProcessorReason, err)
		return utils.K8SUpdateErrorHandler(err) // return error of setting status, or nil if success (since the original error is not retryable and logged)
	}

	err = r.Patch(ctx, processor, client.Apply, client.FieldOwner(action.Name), client.ForceOwnership)
	if err != nil {
		statusErr := r.reportReconciledToProcessorFailed(ctx, action, FailedToCreateProcessorReason, err)
		if statusErr == nil {
			return ctrl.Result{}, err // return original error on success
		} else {
			logger := ctrl.LoggerFrom(ctx)
			logger.Error(statusErr, "Failed to set status on action")
			return ctrl.Result{}, err // return original error on success
		}
	}

	err = r.reportReconciledToProcessor(ctx, action)
	return utils.K8SUpdateErrorHandler(err) // return error of setting status, or nil if success (since the original reconcile is successful)
}
