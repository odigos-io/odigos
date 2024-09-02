package instrumentationconfig

import (
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	rulesv1alpha1 "github.com/odigos-io/odigos/api/rules/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PayloadCollectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *PayloadCollectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	payloadCollectionRules := &rulesv1alpha1.PayloadCollectionList{}
	err := r.Client.List(ctx, payloadCollectionRules)
	if err != nil {
		return ctrl.Result{}, err
	}

	// filter out only enabled rules
	enabledRules := make([]rulesv1alpha1.PayloadCollection, 0, len(payloadCollectionRules.Items))
	for _, rule := range payloadCollectionRules.Items {
		if !rule.Spec.Disabled {
			enabledRules = append(enabledRules, rule)
		}
	}

	instrumentedApplications := &odigosv1alpha1.InstrumentedApplicationList{}
	err = r.Client.List(ctx, instrumentedApplications)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, ia := range instrumentedApplications.Items {
		ic := &odigosv1alpha1.InstrumentationConfig{}
		err = r.Client.Get(ctx, client.ObjectKey{Name: ia.Name, Namespace: ia.Namespace}, ic)
		if client.IgnoreNotFound(err) != nil { // might be just deleted by another controller, in which case ignore
			logger.Error(err, "error fetching instrumentation config", "workload", ia.Name)
			return ctrl.Result{}, err
		}

		err := updateInstrumentationConfigForWorkload(ic, &ia, enabledRules)
		if err != nil {
			logger.Error(err, "error updating instrumentation config", "workload", ia.Name)
			continue
		}

		err = r.Client.Update(ctx, ic)
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "error updating instrumentation config", "workload", ia.Name)
			return ctrl.Result{}, err
		}

		logger.V(0).Info("Updated instrumentation config", "workload", ia.Name)
	}

	logger.V(0).Info("Payload Collection Rules changed, recalculating instrumentation configs", "number of enabled rules", len(enabledRules), "number of instrumented workloads", len(instrumentedApplications.Items))
	return ctrl.Result{}, nil
}
