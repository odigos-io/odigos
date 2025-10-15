package instrumentationconfig

import (
	"context"
	"errors"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
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
	// Verify custom instrumentations if they exist
	for _, rule := range instrumentationRules.Items {
		var err error
		if rule.Spec.CustomInstrumentations != nil {
			if err = rule.Spec.CustomInstrumentations.Verify(); err != nil {
				logger.Error(err, "invalid custom instrumentations", "rule", rule.Name)
			}
		}

		// We join the errors of each rule validation k8 api update
		statusUpdateErr = errors.Join(statusUpdateErr, irc.reportRuleValidationStatus(ctx, &rule, err))
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

	for _, ic := range instrumentationConfigs.Items {
		currIc := ic
		err = updateInstrumentationConfigForWorkload(&currIc, instrumentationRules)
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

func (irc *InstrumentationRuleReconciler) reportRuleValidationStatus(ctx context.Context, ir *odigosv1alpha1.InstrumentationRule, err error) error {
	var (
		condition metav1.ConditionStatus
		reason    string
		message   string
	)
	if err == nil {
		condition = metav1.ConditionTrue
		reason = "VerificationSucceeded"
		message = "Successfuly verified instrumentation rule"
	} else {
		condition = metav1.ConditionFalse
		reason = "VerificationFailed"
		message = err.Error()
	}

	changed := meta.SetStatusCondition(&ir.Status.Conditions, metav1.Condition{
		Type:               odigosv1alpha1.InstrumentationRuleVerified,
		Status:             condition,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: ir.Generation,
	})

	if changed {
		err := irc.Status().Update(ctx, ir)
		if err != nil {
			return err
		}
	}
	return nil
}
