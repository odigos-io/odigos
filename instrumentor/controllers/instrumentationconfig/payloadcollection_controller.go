package instrumentationconfig

import (
	"context"

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

	workloads, err := getAllInstrumentedWorkloads(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, workload := range workloads {
		err = calcInstrumentationConfigForWorkload(workload, enabledRules)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.V(0).Info("Payload Collection Rules changed, recalculating instrumentation configs", "number of enabled rules", len(enabledRules), "number of instrumented workloads", len(workloads))
	return ctrl.Result{}, nil
}
