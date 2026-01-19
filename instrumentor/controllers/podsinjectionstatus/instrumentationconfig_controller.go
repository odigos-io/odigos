package podsinjectionstatus

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InstrumentationConfigController struct {
	client.Client
	PodsTracker *PodsTracker
}

func (r *InstrumentationConfigController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(req.Name, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = syncWorkload(ctx, r.Client, pw)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}

	return ctrl.Result{}, nil
}
