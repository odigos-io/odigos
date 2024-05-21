package instrumentationdevice

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type OdigosConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// odigos config is used to determine the default SDKs to use for each language
// when the config changes (for example, when upgrading from community to cloud tier)
// we may need to re-calculate the instrumentation devices for existing workloads
func (r *OdigosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// this might be overkill, but updating the odigos configuration should be very rare
	// and we want to make sure we don't miss any instrumentation
	// so we'll just re-instrument all instrumented applications
	var instrumentedApplications odigosv1.InstrumentedApplicationList
	err := r.Client.List(ctx, &instrumentedApplications)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconciling all instrumented applications on odigos config change", "numInstrumentedApplications", len(instrumentedApplications.Items))

	for _, instrumentedApplication := range instrumentedApplications.Items {
		err = reconcileSingleInstrumentedApplication(ctx, r.Client, &instrumentedApplication)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
