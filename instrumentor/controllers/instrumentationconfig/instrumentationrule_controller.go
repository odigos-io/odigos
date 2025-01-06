package instrumentationconfig

import (
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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

	instrumentationConfigs := &odigosv1alpha1.InstrumentationConfigList{}
	err = r.Client.List(ctx, instrumentationConfigs)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, ic := range instrumentationConfigs.Items {
		currIc := ic
		workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(ic.Name)
		if err != nil {
			return ctrl.Result{}, err
		}

		serviceName, err := resolveServiceName(ctx, r.Client, workloadName, ic.Namespace, workloadKind)
		if err != nil {
			logger.Error(err, "error resolving service name", "workload", ic.Name)
			continue
		}

		err = updateInstrumentationConfigForWorkload(&currIc, instrumentationRules, serviceName)
		if err != nil {
			logger.Error(err, "error updating instrumentation config", "workload", ic.Name)
			continue
		}

		err = r.Client.Update(ctx, &currIc)
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "error updating instrumentation config", "workload", ic.Name)
			return ctrl.Result{}, err
		}

		logger.V(0).Info("Updated instrumentation config", "workload", ic.Name)
	}

	logger.V(0).Info("Payload Collection Rules changed, recalculating instrumentation configs", "number of instrumentation rules", len(instrumentationRules.Items), "number of instrumented workloads", len(instrumentationConfigs.Items))
	return ctrl.Result{}, nil
}
