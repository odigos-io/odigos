package podsinjectionstatus

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PodsController struct {
	Client      client.Client
	PodsTracker *PodsTracker
}

func (r *PodsController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var pod corev1.Pod
	err := r.Client.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err := r.handleDeletedPod(ctx, req)
			return utils.K8SUpdateErrorHandler(err)
		}
		return ctrl.Result{}, err
	}

	pw, err := workload.PodWorkloadObject(ctx, &pod)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Pod does not have a workload owner (e.g., system pods), skip processing.
	// we only care about pods that are managed by a workload odigos supports.
	if pw == nil {
		return ctrl.Result{}, nil
	}

	err = r.PodsTracker.SetPodWorkload(req, *pw)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Error(err, "error setting pod id to workload mapping in pods tracker", "pod", req.NamespacedName)
	}

	err = syncWorkload(ctx, r.Client, *pw)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PodsController) handleDeletedPod(ctx context.Context, req ctrl.Request) error {
	pw, ok := r.PodsTracker.GetPodWorkload(req)
	if !ok {
		return nil
	}

	err := syncWorkload(ctx, r.Client, pw)
	if err != nil {
		return err
	}

	r.PodsTracker.DeletePodWorkload(req)
	return nil
}
