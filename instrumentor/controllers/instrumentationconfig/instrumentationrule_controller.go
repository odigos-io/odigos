package instrumentationconfig

import (
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type InstrumentationRuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *InstrumentationRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	instrumentationRules := &odigosv1alpha1.InstrumentationRuleList{}
	err := r.Client.List(ctx, instrumentationRules)
	if err != nil {
		return ctrl.Result{}, err
	}

	instrumentedApplications := &odigosv1alpha1.InstrumentedApplicationList{}
	err = r.Client.List(ctx, instrumentedApplications)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, ia := range instrumentedApplications.Items {
		workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(ia.Name)
		if err != nil {
			return ctrl.Result{}, err
		}

		serviceName, err := resolveServiceName(ctx, r.Client, workloadName, ia.Namespace, workloadKind)
		if err != nil {
			logger.Error(err, "error resolving service name", "workload", ia.Name)
			continue
		}
		ic := &odigosv1alpha1.InstrumentationConfig{}
		err = r.Client.Get(ctx, client.ObjectKey{Name: ia.Name, Namespace: ia.Namespace}, ic)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			} else {
				logger.Error(err, "error fetching instrumentation config", "workload", ia.Name)
				return ctrl.Result{}, err
			}
		}

		err = updateInstrumentationConfigForWorkload(ic, &ia, instrumentationRules, serviceName)
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

	logger.V(0).Info("Payload Collection Rules changed, recalculating instrumentation configs", "number of instrumentation rules", len(instrumentationRules.Items), "number of instrumented workloads", len(instrumentedApplications.Items))
	return ctrl.Result{}, nil
}
