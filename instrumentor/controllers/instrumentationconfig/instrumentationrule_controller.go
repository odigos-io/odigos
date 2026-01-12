package instrumentationconfig

import (
	"context"
	"errors"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstrumentationRuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (irc *InstrumentationRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	instrumentationRules := &odigosv1alpha1.InstrumentationRuleList{}
	err := irc.Client.List(ctx, instrumentationRules)
	if err != nil {
		return ctrl.Result{}, err
	}
	var statusUpdateErr error

	// We accumulate valid instrumentation rules here so that only these are applied to instrumentation configs
	validRules := []odigosv1alpha1.InstrumentationRule{}

	// Verify custom instrumentations if they exist
	for _, rule := range instrumentationRules.Items {
		var validationErr error
		if validationErr = rule.Spec.Verify(); validationErr != nil {
			logger.Error(validationErr, "invalid instrumentation rule", "rule", rule.Name)
		} else {
			validRules = append(validRules, rule)
		}

		// write to the rule status on either a successful or un successful verification.
		// join all the status update(k8) errors for requeue if failed to update the status.
		statusUpdateErr = errors.Join(statusUpdateErr, irc.reportRuleValidationStatus(ctx, &rule, validationErr))
	}
	// if the k8 api server errored, we return here such that the instrumentation rule change will get requeued
	if statusUpdateErr != nil {
		return ctrl.Result{}, statusUpdateErr
	}

	instrumentationConfigs := &odigosv1alpha1.InstrumentationConfigList{}
	err = irc.Client.List(ctx, instrumentationConfigs)
	if err != nil {
		return ctrl.Result{}, err
	}

	conf, err := utils.GetCurrentOdigosConfiguration(ctx, irc.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Apply only the valid rules to the instrumentation configs
	instrumentationRules.Items = validRules
	for _, ic := range instrumentationConfigs.Items {
		currIc := ic
		err = updateInstrumentationConfigForWorkload(&currIc, instrumentationRules, &conf)
		if err != nil {
			logger.Error(err, "error updating instrumentation config", "workload", ic.Name)
			continue
		}
		err = irc.Client.Update(ctx, &currIc)
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "error updating instrumentation config", "workload", ic.Name)
			return ctrl.Result{}, err
		}

		logger.V(0).Info("Updated instrumentation config", "workload", ic.Name)
	}

	logger.V(0).Info("Payload Collection Rules changed, recalculating instrumentation configs", "number of instrumentation rules", len(instrumentationRules.Items), "number of instrumented workloads", len(instrumentationConfigs.Items))
	return ctrl.Result{}, nil
}

// reportRuleValidationStatus updates the status condition of the given InstrumentationRule
// based on the result of validationErr. If validationErr is nil, the rule is marked as verified;
// otherwise, it is marked as failed with the error message.
// it returns any error encountered during the k8 object status update.
func (irc *InstrumentationRuleReconciler) reportRuleValidationStatus(ctx context.Context, ir *odigosv1alpha1.InstrumentationRule, validationErr error) error {
	var (
		condition metav1.ConditionStatus
		reason    string
		message   string
	)
	if validationErr == nil {
		condition = metav1.ConditionTrue
		reason = "VerificationSucceeded"
		message = "successfully verified instrumentation rule"
	} else {
		condition = metav1.ConditionFalse
		reason = "VerificationFailed"
		message = validationErr.Error()
	}

	changed := meta.SetStatusCondition(&ir.Status.Conditions, metav1.Condition{
		Type:               odigosv1alpha1.InstrumentationRuleVerified,
		Status:             condition,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: ir.Generation,
	})
	var updateErr error
	if changed {
		updateErr = irc.Status().Update(ctx, ir)
		if updateErr != nil {
			return updateErr
		}
	}
	// If the status update didn't error - we return nil
	return nil
}
